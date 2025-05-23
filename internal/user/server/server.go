package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go-im/api/access"
	"go-im/api/user"
	"go-im/internal/common/errcode"
	"go-im/internal/common/jwt"
	"go-im/internal/common/protocol"
	"go-im/internal/common/types"
	"go-im/internal/pkg/db"
	"go-im/internal/pkg/kafka"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mjson"
	"go-im/internal/pkg/redis"
	"go-im/internal/pkg/utils"
	"go-im/internal/user/model"
	"go-im/internal/user/repository"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

type Server struct {
	user.UnimplementedUserServer

	redis                 *redis.Redis
	userRepository        *repository.UserRepository
	friendRepository      *repository.FriendRepository
	friendApplyRepository *repository.FriendApplyRepository

	kafkaWriter  *kafka.Writer
	pushCh       chan protocol.PushBody
	accessClient access.AccessClient
}

func NewServer(redis *redis.Redis, db *db.DB, kafkaWriter *kafka.Writer, accessClient access.AccessClient) *Server {
	s := &Server{
		redis:                 redis,
		kafkaWriter:           kafkaWriter,
		userRepository:        repository.NewUserRepository(db),
		friendRepository:      repository.NewFriendRepository(db),
		friendApplyRepository: repository.NewFriendApplyRepository(db),
		accessClient:          accessClient,
		pushCh:                make(chan protocol.PushBody, 2000),
	}
	utils.SafeGo(func() {
		s.consume()
	})
	return s
}

func (s *Server) Connect(ctx context.Context, in *user.ConnectReq) (*user.ConnectResp, error) {
	key := fmt.Sprintf(types.CacheOnlineKey, in.UserId)
	_, err := s.redis.Wrap(ctx, func(ctx2 context.Context) (any, string, error) {
		cmd := s.redis.Set(ctx2, key, "", 60*time.Second)
		return cmd.Val(), cmd.String(), cmd.Err()
	})
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	return &user.ConnectResp{}, nil
}

func (s *Server) DisConnect(ctx context.Context, in *user.DisConnectReq) (*user.DisConnectResp, error) {
	key := fmt.Sprintf(types.CacheOnlineKey, in.UserId)
	_, err := s.redis.Wrap(ctx, func(ctx2 context.Context) (any, string, error) {
		cmd := s.redis.Del(ctx2, key)
		return cmd.Val(), cmd.String(), cmd.Err()
	})
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	return &user.DisConnectResp{}, nil
}

func (s *Server) Heartbeat(ctx context.Context, in *user.HeartBeatReq) (*user.HeartBeatResp, error) {
	key := fmt.Sprintf(types.CacheOnlineKey, in.UserId)
	_, err := s.redis.Wrap(ctx, func(ctx2 context.Context) (any, string, error) {
		cmd := s.redis.Expire(ctx2, key, 60*time.Second)
		return cmd.Val(), cmd.String(), cmd.Err()
	})
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	return &user.HeartBeatResp{}, nil
}

func (s *Server) Login(ctx context.Context, in *user.LoginReq) (*user.LoginResp, error) {
	usr, err := s.userRepository.FindOneByPhone(ctx, in.Phone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ToRpcError(errcode.ErrUserNotExists)
		}
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}

	sha1Password := sha256.Sum256([]byte(in.Password + "salt"))
	sha1PasswordStr := hex.EncodeToString(sha1Password[:])
	if sha1PasswordStr != usr.Password {
		return nil, errcode.ToRpcError(errcode.ErrPasswordWrong)
	}
	token, err := jwt.GenerateToken(usr.ID)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}

	return &user.LoginResp{
		Token: token,
	}, nil
}

func (s *Server) Register(ctx context.Context, in *user.RegisterReq) (*user.RegisterResp, error) {
	_, err := s.userRepository.FindOneByPhone(ctx, in.Phone)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("err: %v", err)
			return nil, errcode.ToRpcError(err)
		} else {
			sha1Password := sha256.Sum256([]byte(in.Password + "salt"))
			sha1PasswordStr := hex.EncodeToString(sha1Password[:])
			newUser := &model.Users{
				Username: in.Username,
				Phone:    in.Phone,
				Password: sha1PasswordStr,
				Avatar:   in.Avatar,
				Gender:   in.Gender,
			}
			_, err = s.userRepository.Insert(ctx, newUser)
			if err != nil {
				log.Errorf("err: %v", err)
				return nil, errcode.ToRpcError(err)
			}
			return &user.RegisterResp{}, nil
		}
	}
	return &user.RegisterResp{}, errcode.ToRpcError(errcode.ErrPhoneRegisted)
}

func (s *Server) UpdateInfo(ctx context.Context, in *user.UpdateInfoReq) (*user.UpdateInfoResp, error) {
	usr, err := s.userRepository.FindOne(ctx, int64(in.UserId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &user.UpdateInfoResp{}, errcode.ToRpcError(errcode.ErrUserNotExists)
		}
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}

	if in.Avatar != "" {
		usr.Avatar = in.Avatar
	}
	if in.Username != "" {
		usr.Username = in.Username
	}
	err = s.userRepository.Update(ctx, usr)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}

	onlineUser := make([]int64, 0)
	friends, err := s.friendRepository.ListFriends(ctx, in.UserId)
	if err == nil {
		for _, fd := range friends {
			if s.isUserOnline(ctx, fd.FriendId) {
				onlineUser = append(onlineUser, fd.FriendId)
			}
		}
		if len(onlineUser) > 0 {
			msg := access.FriendUpdatedInfoMsg{
				FriendId: in.UserId,
				ToId:     onlineUser,
			}
			b, _ := mjson.Marshal(&msg)
			s.push(protocol.PushBody{
				Type: protocol.FriendEventTopic,
				Key:  fmt.Appendf([]byte{}, "%d", protocol.FriendInfoUpdatedMsg),
				Body: b,
			})
		}
	}
	return &user.UpdateInfoResp{}, nil
}

func (s *Server) UserInfo(ctx context.Context, in *user.UserInfoReq) (*user.UserInfoResp, error) {
	usr, err := s.userRepository.FindOne(ctx, int64(in.UserId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &user.UserInfoResp{}, errcode.ToRpcError(errcode.ErrUserNotExists)
		}
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	info := &user.UserInfoResp{
		Phone:    usr.Phone,
		Username: usr.Username,
		Avatar:   usr.Avatar,
		UserId:   usr.ID,
		Gender:   usr.Gender,
	}
	return info, nil
}

func (s *Server) DeleteFriend(ctx context.Context, in *user.DeleteFriendReq) (*user.DeleteFriendResp, error) {
	fd, err := s.friendRepository.GetFriendById(ctx, in.UserId, in.FriendId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ToRpcError(errcode.ErrFriendNotExists)
		}
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	err = s.friendRepository.Delete(ctx, uint64(fd.ID))
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	return &user.DeleteFriendResp{}, nil
}

func (s *Server) FriendApply(ctx context.Context, in *user.FriendApplyReq) (*user.FriendApplyResp, error) {
	fd, err := s.friendRepository.GetFriendById(ctx, in.UserId, in.FriendId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("err: %v", err)
			return nil, errcode.ToRpcError(err)
		}
	}
	if fd != nil {
		return nil, errcode.ToRpcError(errcode.ErrFriendExists)
	}
	apply, err := s.friendApplyRepository.GetFriendApply(ctx, in.UserId, in.FriendId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("err: %v", err)
			return nil, errcode.ToRpcError(err)
		}
	}
	if apply != nil {
		return nil, errcode.ToRpcError(errcode.ErrApplyExists)
	}
	apply = &model.FriendApply{
		UserId:   in.UserId,
		FriendId: in.FriendId,
		Status:   repository.FriendApplyStatusPending,
	}
	_, err = s.friendApplyRepository.Insert(ctx, apply)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}

	msg := access.FriendApplyMsg{
		UserId: apply.FriendId,
	}
	b, _ := mjson.Marshal(&msg)
	s.push(protocol.PushBody{
		Type: protocol.FriendEventTopic,
		Key:  fmt.Appendf([]byte{}, "%d", protocol.FriendApplyMsg),
		Body: b,
	})
	return &user.FriendApplyResp{}, nil
}

func (s *Server) HandleApply(ctx context.Context, in *user.HandleApplyReq) (*user.HandleApplyResp, error) {
	apply, err := s.friendApplyRepository.FindOne(ctx, in.ApplyId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ToRpcError(errcode.ErrApplyNotFound)
		}
		return nil, err
	}
	if apply.Status != repository.FriendApplyStatusPending {
		return nil, errcode.ToRpcError(errcode.ErrApplyNotPending)
	}
	if in.Status == repository.FriendApplyStatusAgree {
		err = s.friendApplyRepository.AgreeAndAddFriend(ctx, in.ApplyId, apply.UserId, apply.FriendId)
		if err != nil {
			log.Errorf("err: %v", err)
			return &user.HandleApplyResp{}, errcode.ToRpcError(err)
		}

		msg := access.FriendApplyResponseMsg{
			UserId: apply.UserId,
		}
		b, _ := mjson.Marshal(&msg)
		s.push(protocol.PushBody{
			Type: protocol.FriendEventTopic,
			Key:  fmt.Appendf([]byte{}, "%d", protocol.FriendApplyResultMsg),
			Body: b,
		})
	} else {
		err = s.friendApplyRepository.UpdateFriendApply(ctx, in.ApplyId, repository.FriendApplyStatusReject)
		if err != nil {
			log.Errorf("err: %v", err)
			return &user.HandleApplyResp{}, errcode.ToRpcError(err)
		}
	}
	return &user.HandleApplyResp{}, nil
}

func (s *Server) IsFriend(ctx context.Context, in *user.IsFriendReq) (*user.IsFriendResp, error) {
	_, err := s.friendRepository.GetFriendById(ctx, in.UserId, in.FriendId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &user.IsFriendResp{
				IsFriend: false,
			}, nil
		}
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}

	return &user.IsFriendResp{
		IsFriend: true,
	}, nil
}

func (s *Server) ListApply(ctx context.Context, in *user.ListApplyReq) (*user.ListApplyResp, error) {
	list, err := s.friendApplyRepository.ListFriendApply(ctx, in.UserId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	resp := make([]*user.ApplyInfo, 0)
	for _, apply := range list {
		fdId := 0
		if apply.UserId == in.UserId {
			fdId = int(apply.FriendId)
		} else {
			fdId = int(apply.UserId)
		}
		// TODO 优化批量获取
		fd, err := s.userRepository.FindOne(ctx, int64(fdId))
		if err != nil {
			log.Errorf("err: %v", err)
			return nil, errcode.ToRpcError(err)
		}
		resp = append(resp, &user.ApplyInfo{
			ApplyId:  apply.ID,
			FriendId: apply.FriendId,
			Username: fd.Username,
			Avatar:   fd.Avatar,
			Status:   int32(apply.Status),
			Gender:   fd.Gender,
		})
	}
	return &user.ListApplyResp{
		List: resp,
	}, nil
}

func (s *Server) ListFriends(ctx context.Context, in *user.ListFriendsReq) (*user.ListFriendsResp, error) {
	friends, err := s.friendRepository.ListFriends(ctx, in.UserId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	friendInfo := make([]*user.FriendInfo, 0)
	for _, fd := range friends {
		// TODO 优化批量获取
		usr, err := s.userRepository.FindOne(ctx, int64(fd.FriendId))
		if err != nil {
			log.Errorf("err: %v", err)
			return nil, errcode.ToRpcError(err)
		}
		friendInfo = append(friendInfo, &user.FriendInfo{
			UserId:   usr.ID,
			Username: usr.Username,
			Avatar:   usr.Avatar,
			Gender:   usr.Gender,
			Phone:    usr.Phone,
			Remark:   fd.Remark,
		})
	}
	return &user.ListFriendsResp{
		List: friendInfo,
	}, nil
}

func (s *Server) SearchUser(ctx context.Context, in *user.SearchUserReq) (*user.SearchUserResp, error) {
	usr, err := s.userRepository.FindOneByPhone(ctx, in.Phone)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	list := []*user.SearchUserInfo{
		{
			Id:       usr.ID,
			Username: usr.Username,
			Avatar:   usr.Avatar,
			Gender:   usr.Gender,
			Phone:    usr.Phone,
		},
	}
	return &user.SearchUserResp{
		List: list,
	}, nil
}

func (s *Server) UpdateFriendInfo(ctx context.Context, in *user.UpdateFriendInfoReq) (*user.UpdateFriendInfoResp, error) {
	if in.Remark != "" {
		return nil, nil
	}
	err := s.friendRepository.UpdateFriendInfo(ctx, &model.Friends{
		UserId:   in.UserId,
		FriendId: in.FriendId,
		Remark:   in.Remark,
	})
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	return &user.UpdateFriendInfoResp{}, nil
}

func (s *Server) isUserOnline(ctx context.Context, userId int64) bool {
	key := fmt.Sprintf(types.CacheOnlineKey, userId)
	ret, err := s.redis.Wrap(ctx, func(ctx context.Context) (any, string, error) {
		cmd := s.redis.Exists(ctx, key)
		return cmd.Val(), cmd.String(), cmd.Err()
	})
	if err != nil {
		return false
	}
	return ret.(int64) > 0
}

func (s *Server) push(body protocol.PushBody) {
	s.pushCh <- body
}

func (s *Server) consume() {
	for body := range s.pushCh {
		if s.accessClient != nil {
			_, err := s.accessClient.PushMessage(context.TODO(), &access.PushMessageReq{
				Type: body.Type,
				Key:  body.Key,
				Body: body.Body,
			})
			if err != nil {
				log.Errorf("push rpc message failed, err: %v", err)
			}
		} else {
			s.kafkaWriter.Send(kafkago.Message{
				Topic: body.Type,
				Key:   body.Key,
				Value: body.Body,
			})
		}
	}
}

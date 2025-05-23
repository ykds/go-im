package server

import (
	"context"
	"errors"
	"fmt"
	"go-im/api/access"
	"go-im/api/message"
	"go-im/api/user"
	"go-im/internal/common/errcode"
	"go-im/internal/common/protocol"
	"go-im/internal/common/types"
	"go-im/internal/message/config"
	"go-im/internal/message/model"
	"go-im/internal/message/repository"
	"go-im/internal/pkg/db"
	"go-im/internal/pkg/kafka"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mjson"
	"go-im/internal/pkg/redis"
	"go-im/internal/pkg/utils"
	"strconv"

	kafkago "github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

type Server struct {
	message.UnimplementedMessageServer

	redis                 *redis.Redis
	groupRepository       *repository.GroupRepository
	groupApplyRepository  *repository.GroupApplyRepository
	groupMemberRepository *repository.GroupMemberRepository
	messageRepository     *repository.MessageRepository
	userGroupRepository   *repository.UserGroupRepository
	userSessionRepository *repository.UserSessionRepository

	userRpc      user.UserClient
	accessClient access.AccessClient

	kafkaWriter *kafka.Writer

	pushCh chan protocol.PushBody
}

func NewServer(cfg *config.Config, redis *redis.Redis, db *db.DB, kafkaWriter *kafka.Writer, userRpcClient user.UserClient, accessClient access.AccessClient) *Server {
	s := &Server{
		redis:                 redis,
		kafkaWriter:           kafkaWriter,
		groupRepository:       repository.NewGroupRepository(db),
		groupApplyRepository:  repository.NewGroupApplyRepository(db),
		groupMemberRepository: repository.NewGroupMemberRepository(db),
		messageRepository:     repository.NewMessageRepository(db),
		userGroupRepository:   repository.NewUserGroupRepository(db),
		userSessionRepository: repository.NewUserSessionRepository(db),
		userRpc:               userRpcClient,
		accessClient:          accessClient,
		pushCh:                make(chan protocol.PushBody, 2000),
	}
	utils.SafeGo(func() {
		s.consume()
	})
	return s
}

func (s *Server) AckMessage(ctx context.Context, in *message.AckMessageReq) (*message.AckMessageResp, error) {
	err := s.userSessionRepository.UpdateSessionSeq(ctx, in.SessionId, int64(in.Seq))
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	return &message.AckMessageResp{}, nil
}

func (s *Server) ApplyInGroup(ctx context.Context, in *message.ApplyInGroupReq) (*message.ApplyInGroupResp, error) {
	group, err := s.groupRepository.FindOneByGroupNo(ctx, in.GroupNo)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ToRpcError(errcode.ErrGroupNoExists)
		}
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	isMember, err := s.groupMemberRepository.IsMember(ctx, group.ID, in.UserId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	if isMember {
		return nil, errcode.ToRpcError(errcode.ErrWasGroupMember)
	}
	isApplied, err := s.groupApplyRepository.ApplyExists(ctx, group.ID, in.UserId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	if isApplied {
		return nil, errcode.ToRpcError(errcode.ErrHadApplied)
	}
	apply := model.GroupApply{
		GroupId: group.ID,
		UserId:  in.UserId,
		Status:  repository.GroupApplyWaitStatus,
	}
	_, err = s.groupApplyRepository.Insert(ctx, &apply)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}

	msg := access.GroupApplyMsg{
		UserId:  group.OwnerId,
		GroupId: group.ID,
	}
	b, _ := mjson.Marshal(&msg)
	s.push(protocol.PushBody{
		Type: protocol.GroupEventTopic,
		Key:  fmt.Appendf([]byte{}, "%d", group.OwnerId),
		Body: b,
	})
	return &message.ApplyInGroupResp{}, nil
}

func (s *Server) CreateGroup(ctx context.Context, in *message.CreateGroupReq) (*message.CreateGroupResq, error) {
	no, err := s.genGroupNo(ctx)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	g := model.Group{
		GroupNo: no,
		Name:    in.Name,
		OwnerId: in.UserId,
		Avatar:  in.Avatar,
	}
	id, err := s.groupRepository.CreateGroup(ctx, &g)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	return &message.CreateGroupResq{
		Id:      id,
		GroupNo: g.GroupNo,
		Name:    g.Name,
		Avatar:  g.Avatar,
	}, nil
}

func (s *Server) genGroupNo(ctx context.Context) (int64, error) {
	seq, err := s.redis.Wrap(ctx, func(ctx context.Context) (any, string, error) {
		cmd := s.redis.Incr(ctx, "group_no_seq")
		return cmd.Val(), cmd.String(), cmd.Err()
	})
	return seq.(int64), err
}

func (s *Server) CreateSession(ctx context.Context, in *message.CreateSessionReq) (*message.CreateSessionResp, error) {
	id, err := s.userSessionRepository.Create(ctx, &model.UserSession{
		UserId: in.UserId,
		ToId:   in.FriendId,
		Kind:   "single",
	})
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	if id != 0 {
		key1 := fmt.Sprintf("session:single:%d-%d", in.UserId, in.FriendId)
		_, _ = s.redis.Wrap(ctx, func(ctx context.Context) (any, string, error) {
			cmd := s.redis.Set(ctx, key1, fmt.Sprintf("%d", id), -1)
			return cmd.Val(), cmd.String(), cmd.Err()
		})
	}
	return &message.CreateSessionResp{
		SessionId: id,
	}, nil
}

func (s *Server) DeleteUserSession(ctx context.Context, in *message.DeleteUserSessionReq) (*message.DeleteUserSessionResp, error) {
	// TODO 标记为删除即可，再次发起聊天时重新打开
	err := s.userSessionRepository.Delete(ctx, in.SessionId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	return &message.DeleteUserSessionResp{}, nil
}

func (s *Server) DismissGroup(ctx context.Context, in *message.DismissGroupReq) (*message.DismissGroupResp, error) {
	group, err := s.groupRepository.FindOne(ctx, in.GroupId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	if group.OwnerId != in.UserId {
		return nil, errcode.ToRpcError(errcode.ErrGroupOwnerOnly)
	}
	err = s.groupRepository.DismissGroup(ctx, in.GroupId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}

	onlineUser := make([]int64, 0)
	members, err := s.groupMemberRepository.ListMember(ctx, group.ID)
	if err == nil {
		for _, member := range members {
			if s.isUserOnline(ctx, member.UserId) {
				onlineUser = append(onlineUser, member.UserId)
			}
		}
		if len(onlineUser) > 0 {
			msg := access.GroupUpdatedInfoMsg{
				GroupId: group.ID,
				ToId:    onlineUser,
			}
			b, _ := mjson.Marshal(&msg)
			s.push(protocol.PushBody{
				Type: protocol.GroupEventTopic,
				Key:  fmt.Appendf([]byte{}, "%d", protocol.GroupDismissMsg),
				Body: b,
			})
		}
	}
	return &message.DismissGroupResp{}, nil
}

func (s *Server) ExitGroup(ctx context.Context, in *message.ExitGroupReq) (*message.ExitGroupResp, error) {
	isMember, err := s.groupMemberRepository.IsMember(ctx, in.GroupId, in.UserId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	if !isMember {
		return nil, errcode.ToRpcError(errcode.ErrNotGroupMember)
	}
	err = s.groupMemberRepository.RemvoeMember(ctx, in.GroupId, in.UserId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}

	onlineUser := make([]int64, 0)
	members, err := s.groupMemberRepository.ListMember(ctx, in.GroupId)
	if err == nil {
		for _, member := range members {
			if s.isUserOnline(ctx, member.UserId) {
				onlineUser = append(onlineUser, member.UserId)
			}
		}
		if len(onlineUser) > 0 {
			msg := access.GroupUpdatedInfoMsg{
				GroupId: in.GroupId,
				ToId:    onlineUser,
			}
			b, _ := mjson.Marshal(&msg)
			s.push(protocol.PushBody{
				Type: protocol.GroupEventTopic,
				Key:  fmt.Appendf([]byte{}, "%d", protocol.GroupMemberChangeMsg),
				Body: b,
			})
		}
	}
	return &message.ExitGroupResp{}, nil
}

func (s *Server) HandleGroupApply(ctx context.Context, in *message.HandleGroupApplyReq) (*message.HandleGroupApplyResp, error) {
	apply, err := s.groupApplyRepository.FindOne(ctx, in.ApplyId)
	if err != nil {
		log.Errorf("err: %v", err)
		return &message.HandleGroupApplyResp{}, errcode.ToRpcError(err)
	}
	if apply.Status != repository.GroupApplyWaitStatus {
		return nil, errcode.ToRpcError(errcode.ErrApplyHandled)
	}
	err = s.groupApplyRepository.HandleApply(ctx, in.ApplyId, in.Status)
	if err != nil {
		log.Errorf("err: %v", err)
		return &message.HandleGroupApplyResp{}, errcode.ToRpcError(err)
	}

	msg := access.GroupApplyResponseMsg{
		UserId: apply.UserId,
		Status: in.Status,
	}
	b, _ := mjson.Marshal(&msg)
	s.push(protocol.PushBody{
		Type: protocol.GroupEventTopic,
		Key:  fmt.Appendf([]byte{}, "%d", protocol.GroupAppluResultMsg),
		Body: b,
	})
	return &message.HandleGroupApplyResp{}, nil
}

func (s *Server) InviteMember(ctx context.Context, in *message.InviteMemberReq) (*message.InviteMemberResp, error) {
	isMember, err := s.groupMemberRepository.IsMember(ctx, in.GroupId, in.UserId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	if isMember {
		return nil, errcode.ToRpcError(errcode.ErrWasGroupMember)
	}
	err = s.groupMemberRepository.InviteMember(ctx, in.GroupId, in.InvitedIds)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}

	onlineUser := make([]int64, 0)
	members, err := s.groupMemberRepository.ListMember(ctx, in.GroupId)
	if err == nil {
		for _, member := range members {
			if s.isUserOnline(ctx, member.UserId) {
				onlineUser = append(onlineUser, member.UserId)
			}
		}
		if len(onlineUser) > 0 {
			msg := access.GroupUpdatedInfoMsg{
				GroupId: in.GroupId,
				ToId:    onlineUser,
			}
			b, _ := mjson.Marshal(&msg)

			s.push(protocol.PushBody{
				Type: protocol.GroupEventTopic,
				Key:  fmt.Appendf([]byte{}, "%d", protocol.GroupMemberChangeMsg),
				Body: b,
			})
		}
	}
	return &message.InviteMemberResp{}, nil
}

func (s *Server) ListGroupApply(ctx context.Context, in *message.ListGroupApplyReq) (*message.ListGroupApplyResp, error) {
	group, err := s.groupRepository.ListGroupByOwnerId(ctx, in.UserId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	if len(group) == 0 {
		return &message.ListGroupApplyResp{}, err
	}
	ids := make([]int64, 0, len(group))
	groupMap := make(map[int64]*model.Group, 0)
	for _, g := range group {
		ids = append(ids, g.ID)
		groupMap[g.ID] = g
	}
	apply, err := s.groupApplyRepository.ListApplyByGroupId(ctx, ids)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	if len(apply) == 0 {
		return &message.ListGroupApplyResp{}, nil
	}
	applyMap := make(map[int64][]*model.GroupApply, 0)
	for _, item := range apply {
		g := applyMap[item.GroupId]
		g = append(g, item)
		applyMap[item.GroupId] = g
	}
	resp := make([]*message.ApplyGroup, 0, len(group))
	for gid, item := range applyMap {
		ag := &message.ApplyGroup{
			Name:   groupMap[gid].Name,
			Avatar: groupMap[gid].Avatar,
		}
		for _, apply := range item {
			user, err := s.userRpc.UserInfo(ctx, &user.UserInfoReq{
				UserId: apply.UserId,
			})
			if err != nil {
				log.Errorf("err: %v", err)
				return nil, errcode.ToRpcError(err)
			}
			ag.Apply = append(ag.Apply, &message.UserApply{
				ApplyId: apply.ID,
				Name:    user.Username,
				Avatar:  user.Avatar,
				Gender:  user.Gender,
			})
		}
		resp = append(resp, ag)
	}
	return &message.ListGroupApplyResp{
		List: resp,
	}, nil
}

func (s *Server) ListGroup(ctx context.Context, in *message.ListGroupReq) (*message.ListGroupResp, error) {
	ids, err := s.groupMemberRepository.ListGroupByUserId(ctx, in.UserId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	groups, err := s.groupRepository.ListGroupById(ctx, ids)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	groupInfo := make([]*message.GroupInfo, 0, len(groups))
	for _, group := range groups {
		members, err := s.groupMemberRepository.ListMember(ctx, group.ID)
		if err != nil {
			log.Errorf("err: %v", err)
			return nil, errcode.ToRpcError(err)
		}

		m := make([]*message.GroupMember, 0, len(members))
		for _, mem := range members {
			info, err := s.userRpc.UserInfo(ctx, &user.UserInfoReq{
				UserId: mem.UserId,
			})
			if err != nil {
				log.Errorf("err: %v", err)
				return nil, errcode.ToRpcError(err)
			}
			m = append(m, &message.GroupMember{
				Id:      mem.UserId,
				Name:    info.Username,
				Avatar:  info.Avatar,
				IsOwner: group.OwnerId == mem.UserId,
			})
		}
		groupInfo = append(groupInfo, &message.GroupInfo{
			Id:      group.ID,
			Name:    group.Name,
			GroupNo: group.GroupNo,
			Avatar:  group.Avatar,
			OwnerId: group.OwnerId,
			Members: m,
		})
	}

	return &message.ListGroupResp{
		Groups: groupInfo,
	}, nil
}

func (s *Server) ListGroupMember(ctx context.Context, in *message.ListGroupMemberReq) (*message.ListGroupMemberResp, error) {
	group, err := s.groupRepository.FindOne(ctx, in.GroupId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	isMember, err := s.groupMemberRepository.IsMember(ctx, in.GroupId, in.UserId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	if !isMember {
		return nil, errcode.ToRpcError(errcode.ErrNotGroupMember)
	}
	members, err := s.groupMemberRepository.ListMember(ctx, in.GroupId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	list := make([]*message.GroupMember, 0, len(members))
	for _, member := range members {
		info, err := s.userRpc.UserInfo(ctx, &user.UserInfoReq{
			UserId: member.UserId,
		})
		if err != nil {
			log.Errorf("err: %v", err)
			return nil, errcode.ToRpcError(err)
		}
		list = append(list, &message.GroupMember{
			Id:        int64(info.UserId),
			Name:      info.Username,
			Avatar:    info.Avatar,
			SessionId: member.SessionId,
			IsOwner:   group.OwnerId == member.UserId,
		})
	}
	return &message.ListGroupMemberResp{
		Members: list,
	}, nil
}

func (s *Server) ListSession(ctx context.Context, in *message.ListSessionReq) (*message.ListSessionResp, error) {
	us, err := s.userSessionRepository.ListUserSession(ctx, in.UserId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	infos := make([]*message.SessionInfo, 0)

	for _, item := range us {
		if item.Kind == "group" {
			group, err := s.groupRepository.FindOne(ctx, item.ToId)
			if err != nil {
				log.Errorf("err: %v", err)
				return nil, errcode.ToRpcError(err)
			}
			count, err := s.groupMemberRepository.MemberCount(ctx, group.ID)
			if err != nil {
				log.Errorf("err: %v", err)
				return nil, errcode.ToRpcError(err)
			}
			infos = append(infos, &message.SessionInfo{
				SessionId:   item.ID,
				Kind:        item.Kind,
				GroupId:     &group.ID,
				GroupName:   &group.Name,
				GroupAvatar: &group.Avatar,
				MemberCount: &count,
				Seq:         item.Seq,
			})
		} else if item.Kind == "single" {
			// TODO 优化批量
			userinfo, err := s.userRpc.UserInfo(ctx, &user.UserInfoReq{
				UserId: item.ToId,
			})
			if err != nil {
				log.Errorf("err: %v", err)
				return nil, errcode.ToRpcError(err)
			}
			uid := int64(userinfo.UserId)
			infos = append(infos, &message.SessionInfo{
				SessionId:    item.ID,
				Kind:         item.Kind,
				FriendId:     &uid,
				FriendName:   &userinfo.Username,
				FriendAvatar: &userinfo.Avatar,
				Seq:          item.Seq,
			})
		}
	}
	return &message.ListSessionResp{
		List: infos,
	}, nil
}

func (s *Server) ListUnReadMessage(ctx context.Context, in *message.ListUnReadMessageReq) (*message.ListUnReadMessageResp, error) {
	var (
		result []*model.Message
		err    error
	)
	if in.Kind == "single" {
		result, err = s.messageRepository.ListUnRead(ctx, in.UserId, in.FromId, in.Seq)
		if err != nil {
			log.Errorf("err: %v", err)
			return nil, errcode.ToRpcError(err)
		}
	} else if in.Kind == "group" {
		result, err = s.messageRepository.ListGroupUnRead(ctx, in.GroupId, in.Seq)
		if err != nil {
			log.Errorf("err: %v", err)
			return nil, errcode.ToRpcError(err)
		}
	}
	infos := make([]*message.MessageInfo, 0, len(result))
	for _, item := range result {
		infos = append(infos, &message.MessageInfo{
			Id:      item.ID,
			Content: item.Content,
			Seq:     item.Seq,
			Kind:    item.Kind,
			FromId:  item.FromId,
		})
	}
	return &message.ListUnReadMessageResp{List: infos}, nil
}

func (s *Server) MoveOutMember(ctx context.Context, in *message.MoveOutMemberReq) (*message.MoveOutMemberResp, error) {
	group, err := s.groupRepository.FindOne(ctx, in.GroupId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	if group.OwnerId != in.OpUserId {
		return nil, errcode.ToRpcError(errcode.ErrGroupOwnerOnly)
	}
	isMember, err := s.groupMemberRepository.IsMember(ctx, in.GroupId, in.UserId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	if !isMember {
		return nil, errcode.ToRpcError(errcode.ErrNotGroupMember)
	}
	err = s.groupMemberRepository.RemvoeMember(ctx, group.ID, in.UserId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}

	onlineUser := make([]int64, 0)
	members, err := s.groupMemberRepository.ListMember(ctx, in.GroupId)
	if err == nil {
		for _, member := range members {
			if s.isUserOnline(ctx, member.UserId) {
				onlineUser = append(onlineUser, member.UserId)
			}
		}
		if len(onlineUser) > 0 {
			msg := access.GroupUpdatedInfoMsg{
				GroupId: in.GroupId,
				ToId:    onlineUser,
			}
			b, _ := mjson.Marshal(&msg)
			s.push(protocol.PushBody{
				Type: protocol.GroupEventTopic,
				Key:  fmt.Appendf([]byte{}, "%d", protocol.GroupMemberChangeMsg),
				Body: b,
			})
		}
	}
	return &message.MoveOutMemberResp{}, nil
}

func (s *Server) SearchGroup(ctx context.Context, in *message.SearchGroupReq) (*message.SearchGroupResp, error) {
	ret, err := s.groupRepository.FindOneByGroupNo(ctx, in.GroupNo)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ToRpcError(errcode.ErrGroupNoExists)
		}
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	members, err := s.groupMemberRepository.ListMember(ctx, ret.ID)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	infos := make([]*message.SearchGroupInfo, 0)
	infos = append(infos, &message.SearchGroupInfo{
		Id:          ret.ID,
		GroupNo:     ret.GroupNo,
		Name:        ret.Name,
		Avatar:      ret.Avatar,
		MemberCount: int32(len(members)),
		OwnerId:     ret.OwnerId,
	})
	return &message.SearchGroupResp{Infos: infos}, nil
}

func (s *Server) SendMessage(ctx context.Context, in *message.SendMessageReq) (*message.SendMessageResp, error) {
	var sessionId int64
	if in.Kind == "group" {
		isMember, err := s.groupMemberRepository.IsMember(ctx, in.ToId, in.UserId)
		if err != nil {
			log.Errorf("err: %v", err)
			return nil, errcode.ToRpcError(err)
		}
		if !isMember {
			return nil, errcode.ToRpcError(errcode.ErrNotGroupMember)
		}
	} else if in.Kind == "single" {
		var err error
		sessionId, err = s.createSessionIfNotExists(ctx, in.UserId, in.ToId)
		if err != nil {
			log.Errorf("err: %v", err)
			return nil, errcode.ToRpcError(err)
		}
	} else {
		return nil, errors.New("not supported kind")
	}

	msg := &model.Message{
		FromId:  in.UserId,
		ToId:    in.ToId,
		Kind:    in.Kind,
		Content: in.Message,
		Seq:     in.Seq,
	}
	msgId, err := s.messageRepository.Insert(ctx, msg)
	if err != nil {
		// if model.IsDuplicateSeq(err) {
		// 	return nil, errcode.ToRpcError(errcode.ErrMessageExists)
		// }
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	if msgId == 0 {
		return nil, errcode.ToRpcError(errcode.ErrCreateMessage)
	}

	if in.Kind == "single" {
		if !s.isUserOnline(ctx, in.ToId) {
			return nil, nil
		}
	}
	msg2 := access.MessageBody{
		Id:        msgId,
		SessionId: sessionId,
		FromId:    msg.FromId,
		ToId:      msg.ToId,
		Content:   msg.Content,
		Seq:       msg.Seq,
		Kind:      msg.Kind,
	}
	b, _ := mjson.Marshal(&msg2)

	s.push(protocol.PushBody{
		Type: protocol.MessageTopic,
		Key:  fmt.Appendf([]byte{}, "%s-%d", msg.Kind, msg.ToId),
		Body: b,
	})
	return &message.SendMessageResp{}, nil
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

func (s *Server) createSessionIfNotExists(ctx context.Context, userId, friendId int64) (int64, error) {
	var sessionId int64
	key1 := fmt.Sprintf("session:single:%d-%d", userId, friendId)
	ret, err := s.redis.Wrap(ctx, func(ctx context.Context) (any, string, error) {
		cmd := s.redis.Get(ctx, key1)
		return cmd.Val(), cmd.String(), cmd.Err()
	})
	if err != nil {
		if err.Error() != "redis: nil" {
			return 0, err
		}
	}
	ex1 := ret.(string)
	if ex1 == "" {
		// TODO 调整为重新打开或者创建
		sessionId, err := s.userSessionRepository.Create(ctx, &model.UserSession{
			UserId: userId,
			ToId:   friendId,
			Kind:   "single",
		})
		if err != nil {
			return 0, nil
		}
		if sessionId == 0 {
			session, err := s.userSessionRepository.GetUserSession(ctx, userId, friendId)
			if err != nil {
				return 0, err
			}
			sessionId = session.ID
		}
		s.redis.Wrap(ctx, func(ctx context.Context) (any, string, error) {
			cmd := s.redis.Set(ctx, key1, fmt.Sprintf("%d", sessionId), -1)
			return cmd.Val(), cmd.String(), cmd.Err()
		})
	}
	key2 := fmt.Sprintf("session:single:%d-%d", friendId, userId)
	ret, err = s.redis.Wrap(ctx, func(ctx context.Context) (any, string, error) {
		cmd := s.redis.Get(ctx, key2)
		return cmd.Val(), cmd.String(), cmd.Err()
	})
	if err != nil {
		if err.Error() != "redis: nil" {
			return 0, err
		}
	}
	ex2 := ret.(string)
	if ex2 == "" {
		sessionId, err = s.userSessionRepository.Create(ctx, &model.UserSession{
			UserId: friendId,
			ToId:   userId,
			Kind:   "single",
		})
		if err != nil {
			return 0, nil
		}
		if sessionId == 0 {
			session, err := s.userSessionRepository.GetUserSession(ctx, friendId, userId)
			if err != nil {
				return 0, err
			}
			sessionId = session.ID
		}
		s.redis.Wrap(ctx, func(ctx context.Context) (any, string, error) {
			cmd := s.redis.Set(ctx, key2, fmt.Sprintf("%d", sessionId), -1)
			return cmd.Val(), cmd.String(), cmd.Err()
		})
	} else {
		sessionId, _ = strconv.ParseInt(ex2, 10, 64)
	}
	return sessionId, nil
}

func (s *Server) UpdateGroupInfo(ctx context.Context, in *message.UpdateGroupInfoReq) (*message.UpdateGroupInfoResp, error) {
	group, err := s.groupRepository.FindOne(ctx, in.GroupId)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}
	if group.OwnerId != in.UserId {
		return nil, errcode.ToRpcError(errcode.ErrGroupOwnerOnly)
	}
	if in.Name != "" {
		group.Name = in.Name
	}
	if in.Avatar != "" {
		group.Avatar = in.Avatar
	}
	err = s.groupRepository.Update(ctx, group)
	if err != nil {
		log.Errorf("err: %v", err)
		return nil, errcode.ToRpcError(err)
	}

	onlineUser := make([]int64, 0)
	members, err := s.groupMemberRepository.ListMember(ctx, group.ID)
	if err == nil {
		for _, member := range members {
			if s.isUserOnline(ctx, member.UserId) {
				onlineUser = append(onlineUser, member.UserId)
			}
		}
		if len(onlineUser) > 0 {
			msg := access.GroupUpdatedInfoMsg{
				GroupId: group.ID,
				ToId:    onlineUser,
			}
			b, _ := mjson.Marshal(&msg)
			s.push(protocol.PushBody{
				Type: protocol.GroupEventTopic,
				Key:  fmt.Appendf([]byte{}, "%d", protocol.GroupInfoUpdatedMsg),
				Body: b,
			})
		}
	}
	return &message.UpdateGroupInfoResp{}, nil
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

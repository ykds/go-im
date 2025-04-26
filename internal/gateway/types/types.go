package types

type AckMessageReq struct {
	SessionId int64 `json:"sessionId"`
	Seq       int64 `json:"seq"`
}

type AckMessageResp struct {
}

type ApplyGroup struct {
	Name   string      `json:"name"`
	Avatar string      `json:"avatar"`
	Apply  []UserApply `json:"apply"`
}

type ApplyInGroupReq struct {
	GroupNo int64 `json:"group_no"`
}

type ApplyInGroupResp struct {
}

type ApplyInfo struct {
	ApplyId  int64  `json:"applyId"`
	UserId   int64  `json:"userId"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Gender   string `json:"gender"`
}

type CreateGroupReq struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type CreateGroupResq struct {
	Id      int64  `json:"id"`
	GroupNo int64  `json:"group_no"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar"`
}

type CreateSessionReq struct {
	FriendId int64 `json:"friend_id"`
}

type CreateSessionResp struct {
	SessionId int64 `json:"session_id"`
}

type DeleteFriendReq struct {
	FriendId int64 `json:"friendId"`
}

type DeleteFriendResp struct {
}

type DismissGroupReq struct {
	GroupId int64 `json:"group_id"`
}

type DismissGroupResp struct {
}

type ExitGroupReq struct {
	GroupId int64 `json:"group_id"`
}

type ExitGroupResp struct {
}

type FriendApplyReq struct {
	FriendId int64 `json:"friendId"`
}

type FriendApplyResp struct {
}

type FriendInfo struct {
	UserId   int64  `json:"userId"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Gender   string `json:"gender"`
	Remark   string `json:"remark"`
}

type GroupInfo struct {
	Id      int64         `json:"id"`
	GroupNo int64         `json:"groupNo"`
	Name    string        `json:"name"`
	Avatar  string        `json:"avatar"`
	OwnerId int64         `json:"ownerId"`
	Members []GroupMember `json:"members"`
}

type GroupMember struct {
	Id     int64  `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type HandleApplyReq struct {
	ApplyId int64 `json:"applyId"`
	Status  int   `json:"status"`
}

type HandleApplyResp struct {
}

type HandleGroupApplyReq struct {
	ApplyId int64  `json:"apply_id"`
	Status  string `json:"status"`
}

type HandleGroupApplyResp struct {
}

type InviteMemberReq struct {
	GroupId   int64   `json:"group_id"`
	InvitedId []int64 `json:"invited_ids"`
}

type InviteMemberResp struct {
}

type ListApplyReq struct {
}

type ListApplyResp struct {
	List []ApplyInfo `json:"list"`
}

type ListFriendsReq struct {
}

type ListFriendsResp struct {
	List []FriendInfo `json:"list"`
}

type ListGroupApplyReq struct {
}

type ListGroupApplyResp struct {
	List []ApplyGroup `json:"list"`
}

type ListGroupMemberReq struct {
	GroupId int64 `form:"group_id"`
}

type ListGroupMemberResp struct {
	Members []GroupMember `json:"members"`
}

type ListGroupReq struct {
}

type ListGroupResp struct {
	Groups []GroupInfo `json:"groups"`
}

type ListSessionReq struct {
}

type ListSessionResp struct {
	List []SessionInfo `json:"list"`
}

type ListUnReadMessageReq struct {
	FromId  int64  `form:"fromId,optional"`
	GroupId int64  `form:"groupId,optional"`
	Seq     int64  `form:"seq"`
	Kind    string `form:"kind"`
}

type ListUnReadMessageResp struct {
	List []MessageInfo `json:"list"`
}

type LoginReq struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type LoginResp struct {
	Token string `json:"token"`
}

type MessageInfo struct {
	Id      int64  `json:"id"`
	Kind    string `json:"kind"`
	Content string `json:"content"`
	Seq     int64  `json:"seq"`
	FromId  int64  `json:"fromId"`
}

type MoveOutMemberReq struct {
	GroupId int64 `json:"group_id"`
	UserId  int64 `json:"user_id"`
}

type MoveOutMemberResp struct {
}

type RegisterReq struct {
	Phone           string `json:"phone"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	Avatar          string `json:"avatar"`
	Gender          string `json:"gender"`
}

type RegisterResp struct {
}

type SearchGroupInfo struct {
	Id          int64  `json:"id"`
	GroupNo     int64  `json:"groupNo"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	MemberCount int32  `json:"memberCount"`
	OwnerId     int64  `json:"owner_id"`
}

type SearchGroupReq struct {
	GroupNo int64 `form:"groupNo"`
}

type SearchGroupResp struct {
	Infos []SearchGroupInfo `json:"infos"`
}

type SearchUserInfo struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
	Phone    string `json:"phone"`
	Gender   string `json:"gender"`
	Avatar   string `json:"avatar"`
}

type SearchUserReq struct {
	Phone string `form:"phone"`
}

type SearchUserResp struct {
	List []SearchUserInfo `json:"list"`
}

type SendMessageReq struct {
	Kind    string `json:"kind"`
	ToId    int64  `json:"toId"`
	Message string `json:"message"`
	Seq     int64  `json:"seq"`
}

type SendMessageResp struct {
}

type SessionInfo struct {
	SessionId    int64  `json:"sessionId"`
	Kind         string `json:"kind"`
	Avatar       string `json:"avatar"`
	GroupId      int64  `json:"groupId"`
	GroupName    string `json:"groupName,omitempty"`
	GroupAvatar  string `json:"groupAvatar,omitempty"`
	MemberCount  int64  `json:"memberCount,omitempty"`
	FriendId     int64  `json:"friendId"`
	FrienName    string `json:"friendName"`
	FriendAvatar string `json:"friendAvatar,omitempty"`
	Seq          int64  `json:"seq"`
}

type UpdateGroupInfoReq struct {
	GroupId int64  `json:"group_id"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar"`
}

type UpdateGroupInfoResp struct {
}

type UpdateInfoReq struct {
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

type UpdateInfoResp struct {
}

type UploadReq struct {
}

type UploadResp struct {
	Url string `json:"url"`
}

type UserApply struct {
	ApplyId int64  `json:"apply_id"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar"`
	Gender  string `json:"gender"`
}

type UserInfoReq struct {
}

type UserInfoResp struct {
	Id       int64  `json:"id"`
	Phone    string `json:"phone"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Gender   string `json:"gender"`
}

type UpdateFriendInfoReq struct {
	FriendId int64  `json:"friend_id"`
	Remark   string `json:"remark"`
}

type UpdateFriendInfoResp struct{}

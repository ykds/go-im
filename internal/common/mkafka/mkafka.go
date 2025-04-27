package mkafka

const (
	MessageTopic     = "message"
	GroupEventTopic  = "group-event"
	FriendEventTopic = "friend-event"
)

const (
	AckMsg       int = 1
	HeartbeatMsg int = 2

	MessageMsg    int = 3
	NewMessageMsg int = 4

	FriendApplyMsg       int = 5
	FriendApplyResultMsg int = 6
	FriendInfoUpdatedMsg int = 7

	GroupApplyMsg        int = 8
	GroupAppluResultMsg  int = 9
	GroupInfoUpdatedMsg  int = 10
	GroupDismissMsg      int = 11
	GroupMemberChangeMsg int = 12
)

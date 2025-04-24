package types

type GetSeqReq struct {
	ToId int64  `form:"to_id"`
	Kind string `form:"kind"`
}

package snowflake

import "github.com/bwmarrin/snowflake"

var node *snowflake.Node

func InitSnowflake() {
	var err error
	node, err = snowflake.NewNode(0)
	if err != nil {
		panic(err)
	}
}

func Generate() int64{
	return node.Generate().Int64()
}

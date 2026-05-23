package snowflake

import (
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	node *snowflake.Node
	once sync.Once
)

// Init initializes the snowflake node with the given node ID (0-1023).
func Init(nodeID int64) error {
	var err error
	once.Do(func() {
		node, err = snowflake.NewNode(nodeID)
	})
	return err
}

// NextID generates a new unique snowflake ID.
func NextID() int64 {
	return node.Generate().Int64()
}

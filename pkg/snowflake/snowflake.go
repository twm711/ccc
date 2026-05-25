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
// Panics if Init has not been called; this is a programmer error.
func NextID() int64 {
	if node == nil {
		panic("snowflake: NextID called before Init")
	}
	return node.Generate().Int64()
}

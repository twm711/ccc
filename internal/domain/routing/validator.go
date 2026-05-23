package routing

import "encoding/json"

var validNodeTypes map[NodeType]bool

func init() {
	validNodeTypes = make(map[NodeType]bool, len(AllNodeTypes))
	for _, nt := range AllNodeTypes {
		validNodeTypes[nt] = true
	}
}

// ValidateGraph validates a FlowGraph for structural correctness.
func ValidateGraph(raw json.RawMessage) (*FlowGraph, error) {
	var g FlowGraph
	if err := json.Unmarshal(raw, &g); err != nil {
		return nil, ErrInvalidGraph
	}

	if len(g.Nodes) == 0 {
		return nil, ErrInvalidGraph
	}

	var startCount, endCount int
	nodeIDs := make(map[string]bool, len(g.Nodes))

	for _, n := range g.Nodes {
		if !validNodeTypes[n.Type] {
			return nil, ErrInvalidNodeType
		}
		nodeIDs[n.ID] = true
		switch n.Type {
		case NodeStart:
			startCount++
		case NodeEnd:
			endCount++
		}
	}

	if startCount != 1 {
		return nil, ErrNoStartNode
	}
	if endCount < 1 {
		return nil, ErrNoEndNode
	}

	// Check all exit targets reference existing nodes
	for _, n := range g.Nodes {
		for _, target := range n.Exits {
			if !nodeIDs[target] {
				return nil, ErrDisconnectedNode
			}
		}
	}

	// Check reachability from start node via BFS
	var startID string
	for _, n := range g.Nodes {
		if n.Type == NodeStart {
			startID = n.ID
			break
		}
	}

	reachable := make(map[string]bool)
	queue := []string{startID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if reachable[current] {
			continue
		}
		reachable[current] = true
		for _, n := range g.Nodes {
			if n.ID == current {
				for _, target := range n.Exits {
					if !reachable[target] {
						queue = append(queue, target)
					}
				}
				break
			}
		}
	}

	if len(reachable) != len(g.Nodes) {
		return nil, ErrDisconnectedNode
	}

	return &g, nil
}

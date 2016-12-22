package blackfish

// The interface
type StatusListener interface {
	OnChange(node *Node, status NodeStatus)
}

var statusListeners = make([]StatusListener, 0, 16)

func AddStatusListener(listener StatusListener) {
	statusListeners = append(statusListeners, listener)
}

func doStatusUpdate(node *Node, status NodeStatus) {
	logfInfo("UPDATE: %s is now %v\n", node.Address(), status)

	for _, sl := range statusListeners {
		sl.OnChange(node, status)
	}
}

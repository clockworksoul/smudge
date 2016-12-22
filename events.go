package blackfish

// StatusListener is the interface that must be implemented to take advantage
// of the cluster member status update notification functionality provided by
// the AddStatusListener() function.
type StatusListener interface {
	// The OnChange() function is called whenever the node is notified of any
	// change in the status of a cluster member.
	OnChange(node *Node, status NodeStatus)
}

var statusListeners = make([]StatusListener, 0, 16)

// AddStatusListener allows the submission of a StatusListener implementation
// whose OnChange() function will be called whenever the node is notified of any
// change in the status of a cluster member.
func AddStatusListener(listener StatusListener) {
	statusListeners = append(statusListeners, listener)
}

func doStatusUpdate(node *Node, status NodeStatus) {
	logfInfo("UPDATE: %s is now %v\n", node.Address(), status)

	for _, sl := range statusListeners {
		sl.OnChange(node, status)
	}
}

package presenter

import "github.com/harakeishi/gopose/pkg/types"

// Presenter is the interface for user-facing CLI output.
type Presenter interface {
	Progress(message string)
	PortConflicts(conflicts []types.PortConflictInfo)
	NetworkConflicts(conflicts []types.NetworkConflictInfo)
	Result(message string)
}

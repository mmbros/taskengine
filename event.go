package taskengine

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type EventType int

const (
	EventNil EventType = iota
	EventStart
	EventSuccess
	EventError
	EventCanceled
)

// String representation of an EventType.
func (t EventType) String() string {
	if t < EventNil || t > EventCanceled {
		return "Invalid"
	}
	strings := []string{
		"Nil",
		"Start",
		"Success",
		"Error",
		"Canceled",
	}
	return strings[t]
}

// Event type contains the informations of the task execution.
// Events objects are emitted by the engine.ExecuteEvents method.
// For each (worker, task) pair, it is emitted a Start event
// followeb by a final event that can be a Success, Canceled or Error event.
// The event.Type() method returns the type of event.
type Event struct {
	Result     Result // nil for Start event
	WorkerID   WorkerID
	WorkerInst int
	Task       Task
	TaskStat   TaskStat
	TimeStart  time.Time
	TimeEnd    time.Time // same as TimeStart for Start event
}

// String returns a representation of an event.
func (e *Event) String() string {
	return fmt.Sprintf("%s[%d] %s%v %s",
		e.WorkerID, e.WorkerInst,
		e.Task.TaskID(), e.TaskStat,
		e.Type())
}

// Type method returns the type of Event.
func (e *Event) Type() EventType {
	// TODO: maybe EventCanceled must consider context.DeadlineExceeded error also.
	if e == nil {
		return EventNil
	}
	if e.Result == nil {
		return EventStart
	}
	if e.Result.Error() == nil {
		return EventSuccess
	}
	if errors.Is(e.Result.Error(), context.Canceled) {
		return EventCanceled
	}
	return EventError
}

// IsFirstSuccessOrLastResult returns true if it is the first success result
// or it is the last result and no success was previously found.
func IsFirstSuccessOrLastResult(e *Event) bool {
	if !IsResult(e) {
		return false
	}
	stat := &e.TaskStat
	if e.Result.Error() == nil {
		// first success
		return stat.Success == 1
	} else {
		// last results and no success was found
		return stat.Todo == 0 && stat.Doing == 0 && stat.Success == 0
	}
}

// IsResultUntilFirstSuccess returns true for all the results
// until the first success (included).
func IsResultUntilFirstSuccess(e *Event) bool {
	if !IsResult(e) {
		return false
	}
	stat := &e.TaskStat
	if e.Result.Error() == nil {
		// first success
		return stat.Success == 1
	} else {
		// it is not a success and no success was found
		return stat.Success == 0
	}
}

// IsSuccessOrError returns true if it is a result and
// it is a success or an error result.
// Return false in case of canceled result.
func IsSuccessOrError(e *Event) bool {
	if !IsResult(e) {
		return false
	}
	err := e.Result.Error()
	return !errors.Is(err, context.Canceled)
}

// IsResult return true if the event has a not nil result
// i.e. not a start event.
func IsResult(e *Event) bool {
	return (e != nil) && (e.Result != nil)
}

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

// Event
type Event struct {
	// Type   EventType
	Task           Task
	Worker         Worker
	Inst           int
	Result         Result // nil for Start event
	Stat           TaskStat
	Timestamp      time.Time
	TimestampStart time.Time
}

func (ev Event) String() string {
	return fmt.Sprintf("%s[%d] %s%v %s",
		ev.Worker.WorkerID, ev.Inst,
		ev.Task.TaskID(), ev.Stat,
		ev.Type())
}

func (ev *Event) Type() EventType {
	if ev == nil {
		return EventNil
	}
	if ev.Result == nil {
		return EventStart
	}
	if ev.Result.Error() == nil {
		return EventSuccess
	}
	if errors.Is(ev.Result.Error(), context.Canceled) {
		return EventCanceled
	}
	return EventError
}

// IsFirstSuccessOrLastResult return true if is is a result and:
// - it is the first success, or
// - it is the last result and no success was found
func (ev *Event) IsFirstSuccessOrLastResult() bool {
	if (ev == nil) || (ev.Result == nil) {
		return false
	}
	stat := &ev.Stat
	if ev.Result.Error() == nil {
		// first success
		return stat.Success == 1
	} else {
		// completed (no more workers) and no success was found
		return stat.Todo == 0 && stat.Doing == 0 && stat.Success == 0
	}
}

// IsResultUntilFirstSuccess return true if it is a result and:
// - it is the first success, or
// - it is not a success and no success was previously found
func (ev *Event) IsResultUntilFirstSuccess() bool {
	if (ev == nil) || (ev.Result == nil) {
		return false
	}
	stat := &ev.Stat
	if ev.Result.Error() == nil {
		// first success
		return stat.Success == 1
	} else {
		// it is not a success and no success was found
		return stat.Success == 0
	}
}

// IsSuccessOrError return true if it is a result and
// it is a success or an error result.
// Return false in case of canceled event.
func (ev *Event) IsSuccessOrError() bool {
	if (ev == nil) || (ev.Result == nil) {
		return false
	}
	err := ev.Result.Error()
	return !errors.Is(err, context.Canceled)
}

// IsResult return true if the event has a not nil result
// i.e. not a start event
func (ev *Event) IsResult() bool {
	return (ev != nil) && (ev.Result != nil)
}

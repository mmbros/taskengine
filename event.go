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

// FirstSuccessOrLastError ..
func (ev Event) FirstSuccessOrLastError() bool {
	if ev.Result != nil {
		success := (ev.Result.Error() == nil)
		stat := ev.Stat
		return (success && stat.Success == 1) || (stat.Completed() && stat.Success == 0)
	}
	return false
}

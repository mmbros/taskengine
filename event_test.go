package taskengine

import (
	"context"
	"errors"
	"testing"
)

func TestEventType_String(t *testing.T) {
	tests := []struct {
		name  string
		etype EventType
		want  string
	}{
		{
			name:  "nil",
			etype: EventNil,
			want:  "Nil",
		},
		{
			name:  "start",
			etype: EventStart,
			want:  "Start",
		},
		{
			name:  "Success",
			etype: EventSuccess,
			want:  "Success",
		},
		{
			name:  "Canceled",
			etype: EventCanceled,
			want:  "Canceled",
		},
		{
			name:  "Error",
			etype: EventError,
			want:  "Error",
		},
		{
			name:  "Invalid < 0",
			etype: -1,
			want:  "Invalid",
		},
		{
			name:  "Invalid > 0",
			etype: 100,
			want:  "Invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.etype.String()
			if got != tt.want {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestEvent_Type(t *testing.T) {
	tests := []struct {
		name  string
		event *Event
		want  EventType
	}{
		{
			name:  "nil",
			event: nil,
			want:  EventNil,
		},
		{
			name:  "start",
			event: &Event{},
			want:  EventStart,
		},
		{
			name:  "success",
			event: &Event{Result: &testingResult{}},
			want:  EventSuccess,
		},
		{
			name:  "canceled",
			event: &Event{Result: &testingResult{Err: context.Canceled}},
			want:  EventCanceled,
		},
		{
			name:  "error",
			event: &Event{Result: &testingResult{Err: errors.New("err")}},
			want:  EventError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.event.Type()
			if got != tt.want {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestEvent_FirstSuccessOrLastError(t *testing.T) {
	tests := []struct {
		name  string
		event *Event
		want  bool
	}{
		{
			name:  "event nil",
			event: nil,
			want:  false,
		},
		{
			name:  "result nil",
			event: &Event{},
			want:  false,
		},
		{
			name: "first success",
			event: &Event{
				Result: testingResult{},
				Stat:   TaskStat{10, 20, 5, 1},
			},
			want: true,
		},
		{
			name: "second success",
			event: &Event{
				Result: testingResult{},
				Stat:   TaskStat{10, 20, 5, 2},
			},
			want: false,
		},
		{
			name: "last error and no success",
			event: &Event{
				Result: testingResult{testingError},
				Stat:   TaskStat{0, 0, 5, 0},
			},
			want: true,
		},
		{
			name: "last error with previous success",
			event: &Event{
				Result: testingResult{testingError},
				Stat:   TaskStat{0, 0, 5, 1},
			},
			want: false,
		},
		{
			name: "not last error - todo",
			event: &Event{
				Result: testingResult{testingError},
				Stat:   TaskStat{1, 0, 5, 0},
			},
			want: false,
		},
		{
			name: "not last error - doing",
			event: &Event{
				Result: testingResult{testingError},
				Stat:   TaskStat{0, 1, 5, 0},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.event.IsFirstSuccessOrLastResult()
			if got != tt.want {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}

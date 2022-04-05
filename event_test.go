package taskengine

import (
	"context"
	"testing"
)

func TestEvent(t *testing.T) {

	tests := []struct {
		name    string
		workers []*Worker
		input   map[string]testCaseTasks
		err     error
	}{
		{
			name: "one worker",
			workers: []*Worker{
				{"w1", 1, workFn},
			},
			input: map[string]testCaseTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, true}},
			},
		},
		{
			name: "one worker two instances",
			workers: []*Worker{
				{"w1", 2, workFn},
			},
			input: map[string]testCaseTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, true}},
			},
		},
		{
			name: "two workers",
			workers: []*Worker{
				{"w1", 1, workFn},
				{"w2", 1, workFn},
			},
			input: map[string]testCaseTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, true}},
				"w2": {{"t3", 20, true}, {"t2", 10, true}},
			},
		},
		{
			name: "three workers",
			workers: []*Worker{
				{"w1", 1, workFn},
				{"w2", 1, workFn},
				{"w3", 1, workFn},
			},
			input: map[string]testCaseTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, true}},
				"w2": {{"t3", 20, false}, {"t2", 10, true}},
				"w3": {{"t3", 10, true}},
			},
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := newTestWorkeridTasks(t, tt.input)
			eng, err := newEngine(ctx, tt.workers, tasks)

			if err != nil {
				t.Errorf("newEngine: unexpected error: %s", err)
			}

			eventc, err := eng.ExecuteEvent(All)
			if err != nil {
				t.Errorf("ExecuteEvent: unexpected error: %s", err)
			}

			for ev := range eventc {
				t.Log(ev)
			}

			t.FailNow()
		})
	}
}

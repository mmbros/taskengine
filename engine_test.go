package taskengine

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewEngine_NilContext(t *testing.T) {
	var ctx context.Context
	_, err := NewEngine(ctx, nil, nil)
	if err == nil {
		t.Errorf("Expecting error, got no error")
	} else {
		errmsg := "nil context"
		if err.Error() != errmsg {
			t.Errorf("Expecting error %q, got %q", errmsg, err)
		}
	}
}

func TestNewEngine(t *testing.T) {

	tests := map[string]struct {
		workers []*Worker
		input   map[string]testingTasks
		err     error
	}{
		"duplicate worker": {
			workers: []*Worker{
				{"w1", 1, testingWorkFn},
				{"w2", 2, testingWorkFn},
				{"w1", 3, testingWorkFn},
			},
			input: map[string]testingTasks{},
			err:   errors.New("duplicate worker: WorkerID=\"w1\""),
		},
		"instances < 1": {
			workers: []*Worker{
				{"w1", 1, testingWorkFn},
				{"w2", 2, testingWorkFn},
				{"w3", 0, testingWorkFn},
			},
			input: map[string]testingTasks{},
			err:   errors.New("instances must be in 1..100 range: WorkerID=\"w3\""),
		},
		"instances > 100": {
			workers: []*Worker{
				{"w1", 1, testingWorkFn},
				{"w2", 2, testingWorkFn},
				{"w3", 101, testingWorkFn},
			},
			input: map[string]testingTasks{},
			err:   errors.New("instances must be in 1..100 range: WorkerID=\"w3\""),
		},
		"ko work function": {
			workers: []*Worker{
				{"w1", 1, testingWorkFn},
				{"w2", 2, nil},
				{"w3", 3, testingWorkFn},
			},
			input: map[string]testingTasks{},
			err:   errors.New("work function cannot be nil: WorkerID=\"w2\""),
		},
		"undefined worker": {
			workers: []*Worker{
				{"w1", 1, testingWorkFn},
				{"w2", 2, testingWorkFn},
				{"w3", 3, testingWorkFn},
			},
			input: map[string]testingTasks{
				"w1":   {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, true}},
				"w000": {{"t3", 10, true}},
				"w2":   {{"t3", 20, true}, {"t2", 10, true}},
			},
			err: errors.New("tasks for undefined worker: WorkerID=\"w000\""),
		},
	}

	ctx := context.Background()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			wts := testingWorkerTasks(tt.input)
			_, err := NewEngine(ctx, tt.workers, wts)

			if tt.err == nil {
				if err != nil {
					t.Errorf("unexpected error %q", err)
				}
			} else {
				// tc.err != nil
				if err == nil {
					t.Errorf("expected error %q, found no error", tt.err)
				} else if err.Error() != tt.err.Error() {
					t.Errorf("expected error %q, found error %q", tt.err, err)
				}
			}
		})
	}
}

func TestEngine_ExecuteEvent_NilEngine(t *testing.T) {
	var eng *Engine
	errmsg := "nil engine"
	_, err := eng.ExecuteEvent()
	if err == nil {
		t.Errorf("expecting error, got no error")
	} else if err.Error() != errmsg {
		t.Errorf("expecting error %q, got error %q", errmsg, err)
	}
}
func TestEngine_ExecuteEvent(t *testing.T) {

	tests := []struct {
		name    string
		workers []*Worker
		input   map[string]testingTasks
		want    []testingEventsGroup
	}{
		{
			name: "one worker",
			workers: []*Worker{
				{"w1", 1, testingWorkFn},
			},
			input: map[string]testingTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, false}},
			},
			want: []testingEventsGroup{
				{{"w1", "t1", EventStart}},
				{{"w1", "t1", EventError}},
				{{"w1", "t2", EventStart}},
				{{"w1", "t2", EventSuccess}},
				{{"w1", "t3", EventStart}},
				{{"w1", "t3", EventSuccess}},
			},
		},
		{
			name: "one worker two instances",
			workers: []*Worker{
				{"w1", 2, testingWorkFn},
			},
			input: map[string]testingTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, false}, {"t1", 10, false}},
			},
			want: []testingEventsGroup{
				{{"w1", "t1", EventStart}, {"w1", "t2", EventStart}},
				{{"w1", "t1", EventError}, {"w1", "t2", EventError}, {"w1", "t3", EventStart}},
				{{"w1", "t3", EventSuccess}},
			},
		},
		{
			name: "three workers same task",
			workers: []*Worker{
				{"w1", 1, testingWorkFn},
				{"w2", 2, testingWorkFn},
				{"w3", 3, testingWorkFn},
			},
			input: map[string]testingTasks{
				"w1": {{"t1", 10, false}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 30, true}},
			},
			want: []testingEventsGroup{
				{{"w1", "t1", EventStart}, {"w2", "t1", EventStart}, {"w3", "t1", EventStart}},
				{{"w1", "t1", EventError}},
				{{"w2", "t1", EventSuccess}},
				{{"w3", "t1", EventCanceled}},
			},
		},

		{
			name: "two workers",
			workers: []*Worker{
				{"w1", 1, testingWorkFn},
				{"w2", 1, testingWorkFn},
			},
			input: map[string]testingTasks{
				"w1": {{"t3", 10, true}, {"t2", 10, true}, {"t1", 10, false}},
				"w2": {{"t3", 10, true}, {"t2", 5, false}},
			},
			want: []testingEventsGroup{
				{{"w1", "t1", EventStart}, {"w2", "t2", EventStart}},
				{{"w1", "t1", EventError}, {"w2", "t2", EventError}, {"w1", "t2", EventStart}, {"w2", "t3", EventStart}},
				{{"w2", "t3", EventSuccess}},
				{{"w1", "t2", EventSuccess}},
				{{"w1", "t3", EventStart}},
				{{"w1", "t3", EventCanceled}},
			},
		},
		{
			name: "three workers",
			workers: []*Worker{
				{"w1", 1, testingWorkFn},
				{"w2", 1, testingWorkFn},
				{"w3", 1, testingWorkFn},
			},
			input: map[string]testingTasks{
				"w1": {{"t3", 30, false}, {"t2", 20, true}, {"t1", 6, true}},
				"w2": {{"t3", 20, false}, {"t2", 8, true}},
				"w3": {{"t3", 10, false}},
			},
			want: []testingEventsGroup{
				{{"w1", "t1", EventStart}, {"w2", "t2", EventStart}, {"w3", "t3", EventStart}},
				{{"w1", "t1", EventSuccess}},
				{{"w1", "t2", EventStart}},
				{{"w2", "t2", EventSuccess}},
				{{"w2", "t3", EventStart}, {"w1", "t2", EventCanceled}},
				{{"w1", "t3", EventStart}},
				{{"w3", "t3", EventError}},
				{{"w2", "t3", EventError}},
				{{"w1", "t3", EventError}},
			},
		},
	}

	// copts := cmp.Options{}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wts := testingWorkerTasks(tt.input)
			eng, err := NewEngine(ctx, tt.workers, wts)

			if err != nil {
				t.Errorf("newEngine: unexpected error: %s", err)
			}

			eventc, err := eng.ExecuteEvent()
			if err != nil {
				t.Errorf("ExecuteEvent: unexpected error: %s", err)
			}

			got := []Event{}
			for ev := range eventc {
				t.Log(ev)
				got = append(got, ev)
			}

			if diff := testingEventsDiff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEngine_Execute_NilEngine(t *testing.T) {
	var eng *Engine
	errmsg := "nil engine"
	_, err := eng.Execute(AllResults)
	if err == nil {
		t.Errorf("expecting error, got no error")
	} else if err.Error() != errmsg {
		t.Errorf("expecting error %q, got error %q", errmsg, err)
	}
}

func TestEngine_Execute_FirstSuccessOrLastResult(t *testing.T) {
	workers := []*Worker{
		{"w1", 1, testingWorkFn},
		{"w2", 1, testingWorkFn},
		{"w3", 1, testingWorkFn},
	}

	tests := map[string]struct {
		input    map[string]testingTasks
		expected []testingResult
	}{
		"all ok": {
			input: map[string]testingTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, true}},
				"w2": {{"t3", 20, true}, {"t2", 14, true}},
				"w3": {{"t3", 18, true}},
			},
			expected: []testingResult{
				{"w1", "t1", nil},
				{"w2", "t2", nil},
				{"w3", "t3", nil},
			},
		},
		"w3-t3 ko": {
			input: map[string]testingTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, true}},
				"w2": {{"t3", 20, true}, {"t2", 15, true}},
				"w3": {{"t3", 10, false}},
			},
			expected: []testingResult{
				{"w1", "t1", nil},
				{"w2", "t2", nil},
				{"w2", "t3", nil},
			},
		},
		"all ko": {
			input: map[string]testingTasks{
				"w1": {{"t3", 30, false}, {"t2", 20, false}, {"t1", 10, false}},
				"w2": {{"t3", 20, false}, {"t2", 14, false}},
				"w3": {{"t3", 18, false}},
			},
			expected: []testingResult{
				{"w1", "t1", testingError},
				{"w1", "t2", testingError},
				{"w1", "t3", testingError},
			},
		},
		"all ok w1": {
			input: map[string]testingTasks{
				"w1": {{"t1", 10, true}, {"t2", 10, true}, {"t3", 10, true}},
				"w2": {{"t2", 40, true}, {"t3", 20, true}},
				"w3": {{"t3", 50, true}},
			},
			expected: []testingResult{
				{"w1", "t1", nil},
				{"w1", "t2", nil},
				{"w1", "t3", nil},
			},
		},
		"all ok w1 w2": {
			input: map[string]testingTasks{
				"w1": {{"t1", 10, true}, {"t2", 10, true}, {"t3", 20, true}},
				"w2": {{"t2", 50, true}, {"t3", 10, true}},
				"w3": {{"t3", 50, true}},
			},
			expected: []testingResult{
				{"w1", "t1", nil},
				{"w1", "t2", nil},
				{"w2", "t3", nil},
			},
		},
		"all ko w1 but t1": {
			input: map[string]testingTasks{
				"w1": {{"t1", 10, true}, {"t2", 10, false}, {"t3", 10, false}},
				"w2": {{"t3", 30, false}, {"t2", 14, true}},
				"w3": {{"t3", 50, true}},
			},
			expected: []testingResult{
				{"w1", "t1", nil},
				{"w2", "t2", nil},
				{"w3", "t3", nil},
			},
		},
		"6 ok": {
			input: map[string]testingTasks{
				"w1": {{"t1", 10, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
				"w2": {{"t1", 10, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
				"w3": {{"t1", 10, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
			},
			expected: []testingResult{
				{"w1", "t1", nil},
				{"w2", "t2", nil},
				{"w3", "t3", nil},
				{"w1", "t4", nil},
				{"w2", "t5", nil},
				{"w3", "t6", nil},
			},
		},
		"6 long": {
			input: map[string]testingTasks{
				"w1": {{"t1", 200, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
				"w2": {{"t1", 10, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
				"w3": {{"t1", 15, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
			},
			expected: []testingResult{
				{"w2", "t2", nil},
				{"w3", "t3", nil},
				{"w2", "t4", nil},
				{"w3", "t5", nil},
				{"w2", "t6", nil},
				{"w3", "t1", nil},
			},
		},
	}

	mode := FirstSuccessOrLastResult
	ctx := context.Background()
	copts := cmp.Options{cmp.Comparer(comparerTestingResult)}

	for title, tt := range tests {
		t.Run(title, func(t *testing.T) {
			wts := testingWorkerTasks(tt.input)
			out := mustExecute(ctx, workers, wts, mode)
			results := []testingResult{}
			for res := range out {
				tres := res.(*testingResult)
				results = append(results, *tres)
			}
			if diff := cmp.Diff(tt.expected, results, copts); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEngine_Execute_UntilFirstSuccess(t *testing.T) {
	workers := []*Worker{
		{"w1", 1, testingWorkFn},
		{"w2", 1, testingWorkFn},
		{"w3", 1, testingWorkFn},
		{"w4", 1, testingWorkFn},
	}

	tests := map[string]struct {
		input    map[string]testingTasks
		expected []testingResult
	}{
		"3 ok": {
			input: map[string]testingTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 10, true}},
			},
			expected: []testingResult{
				{"w3", "t1", nil},
			},
		},
		"1 err + 2 ok": {
			input: map[string]testingTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 10, false}},
			},
			expected: []testingResult{
				{"w3", "t1", testingError},
				{"w2", "t1", nil},
			},
		},
		"2 err + 1 ok": {
			input: map[string]testingTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, false}},
				"w3": {{"t1", 10, false}},
			},
			expected: []testingResult{
				{"w3", "t1", testingError},
				{"w2", "t1", testingError},
				{"w1", "t1", nil},
			},
		},
		"3 err": {
			input: map[string]testingTasks{
				"w1": {{"t1", 30, false}},
				"w2": {{"t1", 20, false}},
				"w3": {{"t1", 10, false}},
			},
			expected: []testingResult{
				{"w3", "t1", testingError},
				{"w2", "t1", testingError},
				{"w1", "t1", testingError},
			},
		},
	}

	mode := UntilFirstSuccess
	ctx := context.Background()
	copts := cmp.Options{cmp.Comparer(comparerTestingResult)}

	for title, tt := range tests {
		t.Run(title, func(t *testing.T) {
			wts := testingWorkerTasks(tt.input)
			out := mustExecute(ctx, workers, wts, mode)
			results := []testingResult{}
			for res := range out {
				tres := res.(*testingResult)
				results = append(results, *tres)
			}
			if diff := cmp.Diff(tt.expected, results, copts); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEngine_Execute_IsSuccessOrError(t *testing.T) {
	workers := []*Worker{
		{"w1", 1, testingWorkFn},
		{"w2", 1, testingWorkFn},
		{"w3", 1, testingWorkFn},
		{"w4", 1, testingWorkFn},
	}

	tests := map[string]struct {
		input    map[string]testingTasks
		expected []testingResult
	}{
		"all ok": {
			input: map[string]testingTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 10, true}},
				"w4": {},
			},
			expected: []testingResult{
				{"w3", "t1", nil},
			},
		},
		"first in error": {
			input: map[string]testingTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 10, false}},
			},
			expected: []testingResult{
				{"w3", "t1", testingError},
				{"w2", "t1", nil},
			},
		},
		"last in error but canceled": {
			input: map[string]testingTasks{
				"w1": {{"t1", 10, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 30, false}},
			},
			expected: []testingResult{
				{"w1", "t1", nil},
			},
		},
	}

	mode := SuccessOrError
	ctx := context.Background()
	copts := cmp.Options{cmp.Comparer(comparerTestingResult)}

	for title, tt := range tests {
		t.Run(title, func(t *testing.T) {
			wts := testingWorkerTasks(tt.input)
			out := mustExecute(ctx, workers, wts, mode)
			results := []testingResult{}
			for res := range out {
				tres := res.(*testingResult)
				results = append(results, *tres)
			}
			if diff := cmp.Diff(tt.expected, results, copts); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEngine_Execute_AllResults(t *testing.T) {
	workers := []*Worker{
		{"w1", 1, testingWorkFn},
		{"w2", 1, testingWorkFn},
		{"w3", 1, testingWorkFn},
		{"w4", 1, testingWorkFn},
	}

	tests := map[string]struct {
		input    map[string]testingTasks
		expected []testingResultsGroup
	}{
		"all ok": {
			input: map[string]testingTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 10, true}},
				"w4": {},
			},
			expected: []testingResultsGroup{
				{{"w3", "t1", nil}},
				{{"w1", "t1", context.Canceled}, {"w2", "t1", context.Canceled}},
			},
		},
		"first in error": {
			input: map[string]testingTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 10, false}},
			},
			expected: []testingResultsGroup{
				{{"w3", "t1", testingError}},
				{{"w2", "t1", nil}},
				{{"w1", "t1", context.Canceled}},
			},
		},
		"last in error but canceled": {
			input: map[string]testingTasks{
				"w1": {{"t1", 10, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 30, false}},
			},
			expected: []testingResultsGroup{
				{{"w1", "t1", nil}},
				{{"w2", "t1", context.Canceled}, {"w3", "t1", context.Canceled}},
			},
		},
	}

	mode := AllResults
	ctx := context.Background()

	for title, tt := range tests {
		t.Run(title, func(t *testing.T) {
			wts := testingWorkerTasks(tt.input)
			out := mustExecute(ctx, workers, wts, mode)
			results := []testingResult{}
			for res := range out {
				results = append(results, *res.(*testingResult))
			}
			if diff := testingResultsDiff(tt.expected, results); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

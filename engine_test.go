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
		{"w1", 1, testingWorkFnX},
		{"w2", 1, testingWorkFnX},
		{"w3", 1, testingWorkFnX},
	}

	tests := map[string]struct {
		input    map[string]testingTasks
		expected []testingResultX
	}{
		"all ok": {
			input: map[string]testingTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, true}},
				"w2": {{"t3", 20, true}, {"t2", 14, true}},
				"w3": {{"t3", 18, true}},
			},
			expected: []testingResultX{
				{"t1", nil}, // w1
				{"t2", nil}, // w2
				{"t3", nil}, // w3
			},
		},
		"w3-t3 ko": {
			input: map[string]testingTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, true}},
				"w2": {{"t3", 20, true}, {"t2", 15, true}},
				"w3": {{"t3", 10, false}},
			},
			expected: []testingResultX{
				{"t1", nil}, // w1
				{"t2", nil}, // w2
				{"t3", nil}, // w2
			},
		},
		"all ko": {
			input: map[string]testingTasks{
				"w1": {{"t3", 30, false}, {"t2", 20, false}, {"t1", 10, false}},
				"w2": {{"t3", 20, false}, {"t2", 10, false}},
				"w3": {{"t3", 10, false}},
			},
			expected: []testingResultX{
				{"t1", testingError}, // w1
				{"t2", testingError}, // w1
				{"t3", testingError}, // w1
			},
		},
		"all ok w1": {
			input: map[string]testingTasks{
				"w1": {{"t1", 10, true}, {"t2", 10, true}, {"t3", 10, true}},
				"w2": {{"t2", 40, true}, {"t3", 20, true}},
				"w3": {{"t3", 50, true}},
			},
			expected: []testingResultX{
				{"t1", nil}, // w1
				{"t2", nil}, // w1
				{"t3", nil}, // w1
			},
		},
		"all ok w1 w2": {
			input: map[string]testingTasks{
				"w1": {{"t1", 10, true}, {"t2", 10, true}, {"t3", 20, true}},
				"w2": {{"t2", 50, true}, {"t3", 10, true}},
				"w3": {{"t3", 50, true}},
			},
			expected: []testingResultX{
				{"t1", nil}, // w1
				{"t2", nil}, // w1
				{"t3", nil}, // w2
			},
		},
		"all ko w1 but t1": {
			input: map[string]testingTasks{
				"w1": {{"t1", 10, true}, {"t2", 10, false}, {"t3", 10, false}},
				"w2": {{"t3", 30, false}, {"t2", 14, true}},
				"w3": {{"t3", 50, true}},
			},
			expected: []testingResultX{
				{"t1", nil}, // w1
				{"t2", nil}, // w2
				{"t3", nil}, // w3
			},
		},
		"6 ok": {
			input: map[string]testingTasks{
				"w1": {{"t1", 10, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
				"w2": {{"t1", 10, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
				"w3": {{"t1", 10, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
			},
			expected: []testingResultX{
				{"t1", nil}, // w1
				{"t2", nil}, // w2
				{"t3", nil}, // w3
				{"t4", nil}, // w1
				{"t5", nil}, // w2
				{"t6", nil}, // w3
			},
		},
		"6 long": {
			input: map[string]testingTasks{
				"w1": {{"t1", 200, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
				"w2": {{"t1", 10, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
				"w3": {{"t1", 15, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
			},
			expected: []testingResultX{
				{"t2", nil}, // w2
				{"t3", nil}, // w3
				{"t4", nil}, // w2
				{"t5", nil}, // w3
				{"t6", nil}, // w2
				{"t1", nil}, // w3
			},
		},
	}

	mode := FirstSuccessOrLastResult
	ctx := context.Background()
	copts := cmp.Options{
		cmp.Comparer(func(x, y testingResultX) bool {
			return x.Err == y.Err && x.Tid == y.Tid
		})}

	for title, tt := range tests {
		t.Run(title, func(t *testing.T) {
			wts := testingWorkerTasks(tt.input)
			out := mustExecute(ctx, workers, wts, mode)
			results := []testingResultX{}
			for res := range out {
				tres := res.(*testingResultX)
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
				{}, // w3
			},
		},
		"1 err + 2 ok": {
			input: map[string]testingTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 10, false}},
			},
			expected: []testingResult{
				{testingError}, // w3
				{},             // w2
			},
		},
		"2 err + 1 ok": {
			input: map[string]testingTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, false}},
				"w3": {{"t1", 10, false}},
			},
			expected: []testingResult{
				{testingError}, // w3
				{testingError}, // w2
				{},             // w1
			},
		},
		"3 err": {
			input: map[string]testingTasks{
				"w1": {{"t1", 30, false}},
				"w2": {{"t1", 20, false}},
				"w3": {{"t1", 10, false}},
			},
			expected: []testingResult{
				{testingError}, // w3
				{testingError}, // w2
				{testingError}, // w1
			},
		},
	}

	mode := UntilFirstSuccess
	ctx := context.Background()
	copts := cmp.Options{cmp.Comparer(func(x, y testingResult) bool { return x.Err == y.Err })}

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
				{},                 // w3
				{context.Canceled}, // w2
				{context.Canceled}, // w1
			},
		},
		"first in error": {
			input: map[string]testingTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 10, false}},
			},
			expected: []testingResult{
				{testingError},     // w3
				{},                 // w2
				{context.Canceled}, // w1
			},
		},
		"last in error but canceled": {
			input: map[string]testingTasks{
				"w1": {{"t1", 10, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 30, false}},
			},
			expected: []testingResult{
				{},                 // w1
				{context.Canceled}, // w2
				{context.Canceled}, // w3
			},
		},
	}

	mode := AllResults
	ctx := context.Background()
	copts := cmp.Options{cmp.Comparer(func(x, y testingResult) bool { return x.Err == y.Err })}

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
				{},                 // w3
				{context.Canceled}, // w2
				{context.Canceled}, // w1
			},
		},
		"first in error": {
			input: map[string]testingTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 10, false}},
			},
			expected: []testingResult{
				{testingError},     // w3
				{},                 // w2
				{context.Canceled}, // w1
			},
		},
		"last in error but canceled": {
			input: map[string]testingTasks{
				"w1": {{"t1", 10, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 30, false}},
			},
			expected: []testingResult{
				{},                 // w1
				{context.Canceled}, // w2
				{context.Canceled}, // w3
			},
		},
	}

	mode := AllResults
	ctx := context.Background()
	copts := cmp.Options{cmp.Comparer(func(x, y testingResult) bool { return x.Err == y.Err })}

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

// func TestEngine_ExecuteEvent(t *testing.T) {
// 	type fields struct {
// 		workers     map[WorkerID]*Worker
// 		widtasks    WorkerTasks
// 		ctx         context.Context
// 		workersList []*Worker
// 	}
// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		want    chan Event
// 		wantErr bool
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			eng := &Engine{
// 				workers:     tt.fields.workers,
// 				widtasks:    tt.fields.widtasks,
// 				ctx:         tt.fields.ctx,
// 				workersList: tt.fields.workersList,
// 			}
// 			got, err := eng.ExecuteEvent()
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("Engine.ExecuteEvent() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("Engine.ExecuteEvent() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

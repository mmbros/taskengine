package taskengine

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNewEngineNilContext(t *testing.T) {
	var ctx context.Context
	_, err := newEngine(ctx, nil, nil)
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

	type testCase struct {
		workers []*Worker
		input   map[string]testCaseTasks
		err     error
	}

	testCases := map[string]testCase{
		"duplicate worker": {
			workers: []*Worker{
				{"w1", 1, workFn},
				{"w2", 2, workFn},
				{"w1", 3, workFn},
			},
			input: map[string]testCaseTasks{},
			err:   errors.New("duplicate worker: WorkerID=\"w1\""),
		},
		"instances < 1": {
			workers: []*Worker{
				{"w1", 1, workFn},
				{"w2", 2, workFn},
				{"w3", 0, workFn},
			},
			input: map[string]testCaseTasks{},
			err:   errors.New("instances must be in 1..100 range: WorkerID=\"w3\""),
		},
		"instances > 100": {
			workers: []*Worker{
				{"w1", 1, workFn},
				{"w2", 2, workFn},
				{"w3", 101, workFn},
			},
			input: map[string]testCaseTasks{},
			err:   errors.New("instances must be in 1..100 range: WorkerID=\"w3\""),
		},
		"ko work function": {
			workers: []*Worker{
				{"w1", 1, workFn},
				{"w2", 2, nil},
				{"w3", 3, workFn},
			},
			input: map[string]testCaseTasks{},
			err:   errors.New("work function cannot be nil: WorkerID=\"w2\""),
		},
		"undefined worker": {
			workers: []*Worker{
				{"w1", 1, workFn},
				{"w2", 2, workFn},
				{"w3", 3, workFn},
			},
			input: map[string]testCaseTasks{
				"w1":   {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, true}},
				"w000": {{"t3", 10, true}},
				"w2":   {{"t3", 20, true}, {"t2", 10, true}},
			},
			err: errors.New("tasks for undefined worker: WorkerID=\"w000\""),
		},
	}

	ctx := context.Background()
	for title, tc := range testCases {
		tasks := newTestWorkeridTasks(t, tc.input)
		_, err := newEngine(ctx, tc.workers, tasks)

		if tc.err == nil {
			if err != nil {
				t.Errorf("%s - Unexpected error %q", title, err)
			}
		} else {
			// tc.err != nil
			if err == nil {
				t.Errorf("%s - Expected error %q, found no error", title, tc.err)
			} else if err.Error() != tc.err.Error() {
				t.Errorf("%s - Expected error %q, found error %q", title, tc.err, err)
			}
		}
	}
}

func TestExecuteNilEngine(t *testing.T) {
	var eng *engine
	errmsg := "nil engine"
	_, err := eng.Execute(All)
	if err == nil {
		t.Errorf("expecting error, got no error")
	} else if err.Error() != errmsg {
		t.Errorf("expecting error %q, got error %q", errmsg, err)
	}
}

func TestExecuteFirstSuccessOrLastError(t *testing.T) {
	workers := []*Worker{
		{"w1", 1, workFn},
		{"w2", 1, workFn},
		{"w3", 1, workFn},
	}

	type testCase struct {
		input    map[string]testCaseTasks
		expected testCaseResults
	}

	testCases := map[string]testCase{
		"all ok": {
			input: map[string]testCaseTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, true}},
				"w2": {{"t3", 20, true}, {"t2", 10, true}},
				"w3": {{"t3", 10, true}},
			},
			expected: testCaseResults{
				{"t1", "w1", true},
				{"t2", "w2", true},
				{"t3", "w3", true},
			},
		},
		"w3-t3 ko": {
			input: map[string]testCaseTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, true}},
				"w2": {{"t3", 20, true}, {"t2", 10, true}},
				"w3": {{"t3", 10, false}},
			},
			expected: testCaseResults{
				{"t1", "w1", true},
				{"t2", "w2", true},
				{"t3", "w2", true},
			},
		},
		"all ko": {
			input: map[string]testCaseTasks{
				"w1": {{"t3", 30, false}, {"t2", 20, false}, {"t1", 10, false}},
				"w2": {{"t3", 20, false}, {"t2", 10, false}},
				"w3": {{"t3", 10, false}},
			},
			expected: testCaseResults{
				{"t1", "w1", false},
				{"t2", "w1", false},
				{"t3", "w1", false},
			},
		},
		"all ok w1": {
			input: map[string]testCaseTasks{
				"w1": {{"t1", 10, true}, {"t2", 10, true}, {"t3", 10, true}},
				"w2": {{"t2", 40, true}, {"t3", 20, true}},
				"w3": {{"t3", 50, true}},
			},
			expected: testCaseResults{
				{"t1", "w1", true},
				{"t2", "w1", true},
				{"t3", "w1", true},
			},
		},
		"all ok w1 w2": {
			input: map[string]testCaseTasks{
				"w1": {{"t1", 10, true}, {"t2", 10, true}, {"t3", 20, true}},
				"w2": {{"t2", 50, true}, {"t3", 10, true}},
				"w3": {{"t3", 50, true}},
			},
			expected: testCaseResults{
				{"t1", "w1", true},
				{"t2", "w1", true},
				{"t3", "w2", true},
			},
		},
		"all ko w1 but t1": {
			input: map[string]testCaseTasks{
				"w1": {{"t1", 10, true}, {"t2", 10, false}, {"t3", 10, false}},
				"w2": {{"t3", 30, false}, {"t2", 10, true}},
				"w3": {{"t3", 50, true}},
			},
			expected: testCaseResults{
				{"t1", "w1", true},
				{"t2", "w2", true},
				{"t3", "w3", true},
			},
		},
		"6 ok": {
			input: map[string]testCaseTasks{
				"w1": {{"t1", 10, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
				"w2": {{"t1", 10, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
				"w3": {{"t1", 10, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
			},
			expected: testCaseResults{
				{"t1", "w1", true},
				{"t2", "w2", true},
				{"t3", "w3", true},
				{"t4", "w1", true},
				{"t5", "w2", true},
				{"t6", "w3", true},
			},
		},
		"6 long": {
			input: map[string]testCaseTasks{
				"w1": {{"t1", 200, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
				"w2": {{"t1", 10, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
				"w3": {{"t1", 15, true}, {"t2", 12, true}, {"t3", 14, true}, {"t4", 10, true}, {"t5", 10, true}, {"t6", 10, true}},
			},
			expected: testCaseResults{
				{"t1", "w3", true},
				{"t2", "w2", true},
				{"t3", "w3", true},
				{"t4", "w2", true},
				{"t5", "w3", true},
				{"t6", "w2", true},
			},
		}}

	mode := FirstSuccessOrLastError
	ctx := context.Background()
	copts := cmp.Options{
		cmpopts.SortSlices(testCaseResultLess),
	}

	for title, tc := range testCases {

		tasks := newTestWorkeridTasks(t, tc.input)

		eng, err := newEngine(ctx, workers, tasks)
		if err != nil {
			t.Fatal(err.Error())
		}
		out, err := eng.Execute(mode)
		if err != nil {
			t.Fatal(err.Error())
		}

		results := testCaseResults{}
		for res := range out {
			tres := res.(*testResult)
			results = append(results, tres.ToTestCaseResult())
		}

		if diff := cmp.Diff(tc.expected, results, copts); diff != "" {
			t.Errorf("%s: mismatch (-want +got):\n%s", title, diff)
		}
	}

}

func TestExecuteUntilFirstSuccess(t *testing.T) {
	workers := []*Worker{
		{"w1", 1, workFn},
		{"w2", 1, workFn},
		{"w3", 1, workFn},
	}

	type testCase struct {
		input    map[string]testCaseTasks
		expected testCaseResults
	}

	testCases := map[string]testCase{
		"3 ok": {
			input: map[string]testCaseTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 10, true}},
			},
			expected: testCaseResults{
				{"t1", "w3", true},
			},
		},
		"1 err + 2 ok": {
			input: map[string]testCaseTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 10, false}},
			},
			expected: testCaseResults{
				{"t1", "w3", false},
				{"t1", "w2", true},
			},
		},
		"2 err + 1 ok": {
			input: map[string]testCaseTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, false}},
				"w3": {{"t1", 10, false}},
			},
			expected: testCaseResults{
				{"t1", "w3", false},
				{"t1", "w2", false},
				{"t1", "w1", true},
			},
		},
		"3 err": {
			input: map[string]testCaseTasks{
				"w1": {{"t1", 30, false}},
				"w2": {{"t1", 20, false}},
				"w3": {{"t1", 10, false}},
			},
			expected: testCaseResults{
				{"t1", "w3", false},
				{"t1", "w2", false},
				{"t1", "w1", false},
			},
		},
	}

	mode := UntilFirstSuccess
	ctx := context.Background()
	copts := cmp.Options{
		cmpopts.SortSlices(testCaseResultLess),
	}

	for title, tc := range testCases {
		tasks := newTestWorkeridTasks(t, tc.input)

		eng, err := newEngine(ctx, workers, tasks)
		if err != nil {
			t.Fatal(err.Error())
		}
		out, err := eng.Execute(mode)
		if err != nil {
			t.Fatal(err.Error())
		}

		results := testCaseResults{}
		for res := range out {
			tres := res.(*testResult)
			results = append(results, tres.ToTestCaseResult())
		}

		if diff := cmp.Diff(tc.expected, results, copts); diff != "" {
			t.Errorf("%s: mismatch (-want +got):\n%s", title, diff)
		}

	}
}

func TestExecuteAll(t *testing.T) {
	workers := []*Worker{
		{"w1", 1, workFn},
		{"w2", 1, workFn},
		{"w3", 1, workFn},
		{"w4", 1, workFn},
	}

	type testCase struct {
		input    map[string]testCaseTasks
		expected testCaseResults
	}

	testCases := map[string]testCase{
		"all ok": {
			input: map[string]testCaseTasks{
				"w1": {{"t1", 30, true}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 10, true}},
				"w4": {},
			},
			expected: testCaseResults{
				{"t1", "w3", true},
				{"t1", "w2", false},
				{"t1", "w1", false},
			},
		},
	}

	mode := All
	ctx := context.Background()
	copts := cmp.Options{
		cmpopts.SortSlices(testCaseResultLess),
	}

	for title, tc := range testCases {
		tasks := newTestWorkeridTasks(t, tc.input)

		out, err := Execute(ctx, workers, tasks, mode)
		if err != nil {
			t.Fatal(err.Error())
		}

		results := testCaseResults{}
		for res := range out {
			tres := res.(*testResult)
			results = append(results, tres.ToTestCaseResult())
		}

		if diff := cmp.Diff(tc.expected, results, copts); diff != "" {
			t.Errorf("%s: mismatch (-want +got):\n%s", title, diff)
		}

	}
}

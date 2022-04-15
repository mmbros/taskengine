package taskengine

import (
	"context"
	"errors"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var testingError error = errors.New("testing error")

// testingTask is used to define a task in the test cases.
type testingTask struct {
	taskid  string // task name
	msec    int    // task duration
	success bool   // task result
}

func (tt *testingTask) TaskID() TaskID { return TaskID(tt.taskid) }

func comparerTestingTask(x, y testingTask) bool {
	return (x.msec == y.msec) && (x.taskid == y.taskid) && (x.success == y.success)
}

// testingTasks is an array of testingTask
type testingTasks []*testingTask

type testingResult struct {
	Wid string
	Tid string
	Err error
}

type testingResultsGroup []testingResult

func (tr testingResult) String() string {
	if tr.Err == nil {
		return "SUCCESS"
	}
	if errors.Is(tr.Err, context.Canceled) {
		return "CANCELED"
	}
	return "ERROR"
}

func (tr testingResult) Error() error { return tr.Err }

func comparerTestingResult(x, y testingResult) bool {
	return (x.Err == y.Err) && (x.Tid == y.Tid) && (x.Wid == y.Wid)
}

// event's informations that will be checked
type testingEvent struct {
	Wid   string
	Tid   string
	Etype EventType
}

type testingEventsGroup []testingEvent

func testingWorkFn(ctx context.Context, worker *Worker, workerInst int, task Task) Result {
	t := task.(*testingTask)
	r := &testingResult{
		Tid: t.taskid,
		Wid: string(worker.WorkerID),
	}

	select {
	case <-ctx.Done():
		r.Err = ctx.Err()
	case <-time.After(time.Duration(t.msec) * time.Millisecond):
		if !t.success {
			r.Err = testingError
		}
	}
	return r
}

func testingTasksToTasks(tts testingTasks) Tasks {
	tasks := Tasks{}
	for _, tt := range tts {
		tasks = append(tasks, tt)
	}
	return tasks
}
func testingWorkerTasks(wts map[string]testingTasks) WorkerTasks {
	wtasks := WorkerTasks{}
	for wid, tcts := range wts {
		wtasks[WorkerID(wid)] = testingTasksToTasks(tcts)
	}
	return wtasks
}

// testingEventsDiff checks if the given events matched the expected events group list.
// The events of the same group can occur in any order.
// If group A is before group B, all the events of group A must precede the events of group B.
// Example:
//     the groups list
//           1|2 3|4 5 6
//     is matched by the events
//           1 2 3 4 5 6
//           1 3 2 4 5 6
//           1 3 2 6 4 5
//     but not by
//           1 2 4 3 5 6
func testingEventsDiff(want []testingEventsGroup, events []Event) string {

	// convert []Event to []testingEvent
	got := []testingEvent{}
	for _, event := range events {
		te := testingEvent{
			Wid:   string(event.WorkerID),
			Tid:   string(event.Task.TaskID()),
			Etype: event.Type(),
		}
		got = append(got, te)
	}

	// lessFunc for testingEvent
	lessFunc := func(x, y testingEvent) bool {
		return (x.Wid < y.Wid) ||
			((x.Wid == y.Wid) && (x.Tid < y.Tid)) ||
			((x.Wid == y.Wid) && (x.Tid == y.Tid) && (x.Etype < y.Etype))
	}
	copts := cmp.Options{cmpopts.SortSlices(lessFunc)}

	// compare each wantGroup with the corrisponding gotGroup
	// The gotGroup contains the same number of element of the wantGroup (if possible)
	// starting from the first not already used element
	curr := 0
	tot := len(got)
	for _, wantGroup := range want {
		L := len(wantGroup)
		if curr+L > tot {
			L = tot - curr
		}

		gotGroup := testingEventsGroup(got[curr : curr+L])
		if diff := cmp.Diff(wantGroup, gotGroup, copts); diff != "" {
			return diff
		}
		curr += L
	}
	if curr < tot {
		wantGroup := testingEventsGroup(nil)
		gotGroup := testingEventsGroup(got[curr:])
		return cmp.Diff(wantGroup, gotGroup, copts)
	}

	return ""
}

// testingResultsDiff is like testingEventsDiff but for results.
func testingResultsDiff(want []testingResultsGroup, got []testingResult) string {

	// lessFunc for testingResult
	lessFunc := func(x, y testingResult) bool {

		if (x.Wid == y.Wid) && (x.Tid == y.Tid) {
			// check error
			var xe, ye string
			if x.Err != nil {
				xe = x.Err.Error()
			}
			if y.Err != nil {
				ye = y.Err.Error()
			}
			return xe < ye
		}

		return (x.Wid < y.Wid) ||
			((x.Wid == y.Wid) && (x.Tid < y.Tid))
	}
	copts := cmp.Options{cmpopts.SortSlices(lessFunc), cmpopts.EquateErrors()}

	// compare each wantGroup with the corrisponding gotGroup
	// The gotGroup contains the same number of element of the wantGroup (if possible)
	// starting from the first not already used element
	curr := 0
	tot := len(got)
	for _, wantGroup := range want {
		L := len(wantGroup)
		if curr+L > tot {
			L = tot - curr
		}

		gotGroup := testingResultsGroup(got[curr : curr+L])
		if diff := cmp.Diff(wantGroup, gotGroup, copts); diff != "" {
			return diff
		}
		curr += L
	}
	if curr < tot {
		wantGroup := testingResultsGroup(nil)
		gotGroup := testingResultsGroup(got[curr:])
		return cmp.Diff(wantGroup, gotGroup, copts)
	}

	return ""
}

func mustExecute(ctx context.Context, workers []*Worker, wts WorkerTasks, mode Mode) chan Result {
	eng, err := NewEngine(workers, wts)
	if err != nil {
		panic("NewEngine:" + err.Error())
	}
	out, err := eng.Execute(ctx, mode)
	if err != nil {
		panic("Execute:" + err.Error())
	}
	return out
}

package taskengine

import (
	"context"
	"errors"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// testingTask is used to define a task in the test cases.
type testingTask struct {
	taskid  string // task name
	msec    int    // task duration
	success bool   // task result
}

func (tt testingTask) TaskID() TaskID { return TaskID(tt.taskid) }

// testingTasks is an array of testingTask
type testingTasks []testingTask

type testingResult struct {
	err error
}

func (tr testingResult) Error() error { return tr.err }

// event's informations that will be checked
type testingEvent struct {
	Wid   string
	Tid   string
	Etype EventType
}

type testingEventsGroup []testingEvent

func testingWorkFn(ctx context.Context, workerInst int, task Task) Result {
	t := task.(testingTask)
	r := &testResult{}

	select {
	case <-ctx.Done():
		r.err = ctx.Err()
	case <-time.After(time.Duration(t.msec) * time.Millisecond):
		if !t.success {
			r.err = errors.New("ERROR")
		}
	}
	return r
}

// testingWorkerTasks creates a WorkerTasks object from a map
//    workerId -> [testingTask1, testingTask2, ...]
// It is useful to shorten the test case declaration
// from
//    workersTasks: WorkerTasks{
//	      "w1": Tasks{
//		      testingTask{"t3", 30, true},
//		      testingTask{"t2", 20, true},
//		      testingTask{"t1", 10, false}}}
// to
//    input: {
//	      "w1": {{"t3", 30, true}, {"t2", 20, true},{"t1", 10, false}}}
func testingWorkerTasks(wts map[string]testingTasks) WorkerTasks {

	wtasks := WorkerTasks{}

	for wid, tcts := range wts {
		ts := Tasks{}
		for _, tct := range tcts {
			ts = append(ts, tct)
		}
		wtasks[WorkerID(wid)] = ts
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
			Wid:   string(event.Worker.WorkerID),
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

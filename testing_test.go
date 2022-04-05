package taskengine

import (
	"context"
	"errors"
	"time"

	"github.com/google/go-cmp/cmp"
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

// event's informations that will be checked
type testingEvent struct {
	Wid   string
	Tid   string
	Etype EventType
}

type testingEventsGroup []testingEvent

func testingEventsDiff(want []testingEventsGroup, got []Event) string {

	for _, event := range got {

		teGot := testingEvent{
			Wid:   string(event.Worker.WorkerID),
			Tid:   string(event.Task.TaskID()),
			Etype: event.Type(),
		}

		// check exists a testingEvent group
		if len(want) == 0 {
			return cmp.Diff(nil, teGot, nil)
		}
		want0 := want[0]

		// check if event is in the first testingEvent group
		found := -1
		for i, te := range want0 {
			if cmp.Diff(te, teGot, nil) == "" {
				found = i
				break
			}
		}
		if found < 0 {
			// event is not found in testingEvent group
			return cmp.Diff(want0, testingEventsGroup{teGot}, nil)
		}

		if len(want0) == 1 {
			// the first group has no more elements
			want = want[1:]
		} else {
			// remove the idx-nth element from the first group
			want0 = append(want0[:found], want0[found+1:]...)
			want[0] = want0
		}

	}
	if len(want) > 0 {
		return cmp.Diff(want, nil, nil)
	}

	return ""
}

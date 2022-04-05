// File common_test.go.

package taskengine

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// testCaseTask is used to define a task in the test cases.
type testCaseTask struct {
	taskid  string // task name
	msec    int    // task duration
	success bool   // task result
}

// testCaseTasks is an aray of testCaseTask
type testCaseTasks []testCaseTask

// testTask is the Task object of the work function.
// It is created from testCaseTask.
type testTask struct {
	testCaseTask
	workerid string
	t        *testing.T
	// iter     int
}

// testResult is the Result returned by the work function.
type testResult struct {
	taskid     string
	workerid   string
	workerinst int
	result     string
	err        error
	// iter       int
}

// testCaseResult is build form testResult.
// It is used to define the expected results in the test cases.
type testCaseResult struct {
	taskid   string
	workerid string
	success  bool
}

// testCaseResults implements sort.Interface for []*testCaseResult
type testCaseResults []*testCaseResult

// interface to pass info from engine execute function to task
// debug porpouses only.
// type headacher interface {
// 	SetIter(int)
// }

// func (t *testTask) SetIter(iter int) { t.iter = iter }

func (t *testCaseTask) TaskID() TaskID                 { return TaskID(t.taskid) }
func (t *testCaseTask) Equal(other *testCaseTask) bool { return t.taskid == other.taskid }
func (t *testCaseTask) String() string                 { return string(t.taskid) }

// func (res *testResult) Success() bool { return res.err == nil }
func (res *testResult) Error() error { return res.err }
func (res *testResult) Status() string {
	if res.err == nil {
		return "SUCCESS"
	}
	return res.err.Error()
}
func (res *testResult) ToTestCaseResult() *testCaseResult {
	return &testCaseResult{
		taskid:   res.taskid,
		workerid: res.workerid,
		success:  res.Error() == nil,
	}
}

func (res *testCaseResult) String() string {
	return fmt.Sprintf("{tid:%q, wid:%q: %v}", res.taskid, res.workerid, res.success)
}
func (res *testCaseResult) Equal(other *testCaseResult) bool {
	return res.taskid == other.taskid &&
		res.workerid == other.workerid &&
		res.success == other.success
}

// testCaseResultLess function is used to sort testCaseResult
// NOTE: it is important to use *simpleResult and not simpleResult
func testCaseResultLess(a, b *testCaseResult) bool {
	return (a.taskid < b.taskid) ||
		(a.taskid == b.taskid && a.workerid < b.workerid) ||
		(a.taskid == b.taskid && a.workerid == b.workerid && !a.success)
}

func (a testCaseResults) Len() int      { return len(a) }
func (a testCaseResults) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a testCaseResults) Less(i, j int) bool {
	return (a[i].taskid < a[j].taskid) ||
		(a[i].taskid == a[j].taskid && a[i].workerid < a[j].workerid)
}

// newTestWorkeridTasks creates a WorkerTasks object from a map workerId -> [testCaseTask1, testCaseTask2, ...]
func newTestWorkeridTasks(t *testing.T, wts map[string]testCaseTasks) WorkerTasks {
	wtasks := WorkerTasks{}

	for wid, tcts := range wts {
		ts := Tasks{}
		for _, tct := range tcts {
			// tt := &testTask{tct, wid, t, 0}
			tt := &testTask{tct, wid, t}
			if tct.msec <= 0 {
				tt.msec = 10
			}
			ts = append(ts, tt)
		}
		wtasks[WorkerID(wid)] = ts
	}
	return wtasks
}

func workFn(ctx context.Context, workerInst int, task Task) Result {

	ttask := task.(*testTask)
	if ttask == nil {
		panic("task is not a testTask: ahhh")
	}

	tres := &testResult{
		taskid:     ttask.taskid,
		workerid:   ttask.workerid,
		workerinst: workerInst,
		result:     fmt.Sprintf("%dms", ttask.msec),
		// iter:       ttask.iter,
	}

	// ttask.t.Logf("WORKING:   (%s, %s, %d)", ttask.workerid, ttask.taskid, ttask.iter)
	// ttask.t.Logf("WORKING:   (%s, %s)", ttask.workerid, ttask.taskid)

	select {
	case <-ctx.Done():
		tres.err = ctx.Err()
	case <-time.After(time.Duration(ttask.msec) * time.Millisecond):
		if !ttask.success {
			tres.err = errors.New("ERR")
		}
	}

	// ttask.t.Logf("WORKED: (%s, %s, %d) -> %s", ttask.workerid, ttask.taskid, ttask.iter, tres.Status())
	// ttask.t.Logf("WORKED: (%s, %s) -> %s", ttask.workerid, ttask.taskid, tres.Status())

	return Result(tres)
}

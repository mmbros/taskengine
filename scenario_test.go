package taskengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"
)

type demoTask struct {
	taskid   string // task name
	workerid string
	rr       *demoRandomResult
	t        *testing.T
}

type demoRandomResult struct {
	msecStdDev float64
	msecMean   float64
	errPerc    int
}

type demoResult struct {
	TaskID     string    `json:"isin"`
	WorkerID   string    `json:"source"`
	WorkerInst int       `json:"instance"`
	Result     string    `json:"result"`
	TimeStart  time.Time `json:"time_start"`
	TimeEnd    time.Time `json:"time_end"`
	ErrMsg     string    `json:"error,omitempty"`
	err        error
}

func demoWorkFn(ctx context.Context, workerInst int, task Task) Result {

	ttask := task.(*demoTask)
	if ttask == nil {
		panic("task is not a demoTask: ahhh")
	}

	msec := ttask.rr.msec()

	tres := &demoResult{
		TaskID:     ttask.taskid,
		WorkerID:   ttask.workerid,
		WorkerInst: workerInst,
		Result:     fmt.Sprintf("%dms", msec),
		TimeStart:  time.Now(),
	}

	// ttask.t.Logf("WORKING: (%s, %s, %dms)", ttask.workerid, ttask.taskid, msec)

	select {
	case <-ctx.Done():
		tres.err = ctx.Err()
	case <-time.After(time.Duration(msec) * time.Millisecond):
		if !ttask.rr.success() {
			tres.err = errors.New("ERR")
		}
	}

	// ttask.t.Logf("WORKED: (%s, %s) -> %s", ttask.workerid, ttask.taskid, tres.Status())

	tres.TimeEnd = time.Now()
	if tres.err != nil {
		tres.ErrMsg = tres.Status()
	}

	return Result(tres)
}

func (rr *demoRandomResult) msec() int64 {
	x := rand.NormFloat64()*rr.msecStdDev + rr.msecMean
	if x < 0 {
		x = 0
	}
	return int64(x)
}

func (rr *demoRandomResult) success() bool {
	n := rand.Intn(101)
	return n > rr.errPerc
}

func (t *demoTask) TaskID() TaskID { return TaskID(t.taskid) }

func (res *demoResult) Success() bool { return res.err == nil }
func (res *demoResult) Status() string {
	if res.err == nil {
		return "SUCCESS"
	}
	return res.err.Error()
}

// scenario return a random Workers and WorkerTasks scenario with the given parameters.
//
// workers: number of workers
// instances: number of instances for each worker
// tasks: number of task
// spread: perc of how many workers executes each tasks:
//         100% - each task is executed by all worker
//           0% - no worker executes the tasks
func scenario(t *testing.T, workers, instances, tasks, spread int, rr *demoRandomResult) ([]*Worker, WorkerTasks) {
	ws := []*Worker{}
	wts := WorkerTasks{}

	for wj := 1; wj <= workers; wj++ {
		wid := WorkerID(fmt.Sprintf("w%d", wj))
		w := &Worker{
			WorkerID:  wid,
			Instances: instances,
			Work:      demoWorkFn,
		}
		ws = append(ws, w)

		ts := Tasks{}
		for tj := 1; tj <= tasks; tj++ {

			n := rand.Intn(101)
			if n <= spread {
				// assign the task to the worker
				tid := fmt.Sprintf("t%d", tj)
				task := &demoTask{
					taskid:   tid,
					workerid: string(wid),
					rr:       rr,
					t:        t,
				}
				ts = append(ts, task)
			}
		}
		wts[wid] = ts
	}

	return ws, wts
}

func TestDemoResult(t *testing.T) {
	rr := &demoRandomResult{
		msecStdDev: 75.0,
		msecMean:   200.0,
		errPerc:    50,
	}

	for j := 0; j < 20; j++ {
		t.Logf("%02d) msec=%d, res=%v\n", j, rr.msec(), rr.success())
	}
	// t.Fail()
}

// TestScenario executes a random scenario with the given parameters.
// The results are saved in "/tmp/quote_demo.json" file.
func TestScenario(t *testing.T) {

	// t.Skip("skipping test")

	rr := &demoRandomResult{
		msecStdDev: 50.0,
		msecMean:   200.0,
		errPerc:    50,
	}
	ws, wts := scenario(t, 5, 2, 100, 90, rr)
	ctx := context.Background()

	out, err := Execute(ctx, ws, wts, All)
	if err != nil {
		t.Fatal(err.Error())
	}

	demoResults := []*demoResult{}
	for res := range out {
		dres := res.(*demoResult)

		if !errors.Is(dres.err, context.Canceled) || (dres.TimeEnd.Sub(dres.TimeStart).Milliseconds() > 10) {
			demoResults = append(demoResults, dres)
		}

	}

	msg, _ := json.MarshalIndent(demoResults, "", " ")

	err = ioutil.WriteFile("/tmp/quote_demo.json", msg, 0644)
	if err != nil {
		t.Fatal(err.Error())
	}
	// fmt.Println(string(msg))

	// t.FailNow()
}

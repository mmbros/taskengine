// Package taskengine concurrently execute a set of tasks assigned to multiple different workers.
//
// Each worker can works all or a subset of the tasks.
//
// When a worker is ready, the next task to execute is dynamically choosen
// considering the current status of the tasks
// so to maximize the thoughput of the tasks successfully executed.
//
// After the first success result is found the remaining jobs for same task are cancelled.
package taskengine

import (
	"context"
	"fmt"
	"time"
)

// Mode of execution for each task.
type Mode int

// Values of mode of execution for each task.
const (
	// For each task returns the result of all the workers: success, error or canceled.
	// Multiple success results can be returned.
	AllResults Mode = iota

	// For each task returns the success or error results.
	// The canceled resuts are not returned.
	// Multiple success results can be returned.
	SuccessOrErrorResults

	// For each task returns the results preceding the first success (included).
	// At most one success is returned.
	ResultsUntilFirstSuccess

	// For each task returns only one result: the first success or the last result.
	FirstSuccessOrLastResult
)

// Engine type is the main struct used to execute the tasks.
// It internally saves the inforations about the workers and the tasks of each worker.
type Engine struct {
	workers     map[WorkerID]*Worker
	widtasks    WorkerTasks // map[WorkerID]*Tasks
	workersList []*Worker   // original workers list
}

// jobInput is the internal struct passed to a worker to execute a task.
type jobInput struct {
	ctx    context.Context    // task context
	cancel context.CancelFunc // func to cancel task execution
	task   Task               // task to be executed
	outc   chan *jobOutput    // output channel
	stat   TaskStat           // used for Start event
}

// jobOutput contains the result returned by the worker with the
// WorkerID and instance in executing the given task.
// A nil result indicates that the worker instance is ready to perform a task.
type jobOutput struct {
	res       Result // can be nil
	wid       WorkerID
	instance  int
	task      Task // not used if res is nil
	timeStart time.Time
	timeEnd   time.Time
}

// NewEngine initialize a new engine object from the list of workers and the tasks of each worker.
// It performs some sanity checks and returns error in case of incongruences.
func NewEngine(ws []*Worker, wts WorkerTasks) (*Engine, error) {

	// check workers and build a map from workerid to Worker
	workers := map[WorkerID]*Worker{}
	for _, w := range ws {
		if _, ok := workers[w.WorkerID]; ok {
			return nil, fmt.Errorf("duplicate worker: WorkerID=%q", w.WorkerID)
		}
		if w.Instances <= 0 || w.Instances > maxInstances {
			return nil, fmt.Errorf("instances must be in 1..%d range: WorkerID=%q", maxInstances, w.WorkerID)
		}
		if w.Work == nil {
			return nil, fmt.Errorf("work function cannot be nil: WorkerID=%q", w.WorkerID)
		}
		workers[w.WorkerID] = w
	}

	// create each taskID context
	widtasks := WorkerTasks{}

	for wid, ts := range wts {
		// for empty task lists, continue
		if len(ts) == 0 {
			continue
		}
		// check the worker exists
		if _, ok := workers[wid]; !ok {
			return nil, fmt.Errorf("tasks for undefined worker: WorkerID=%q", wid)
		}
		// save the task list of the worker in the engine
		widtasks[wid] = ts
	}

	return &Engine{
		workers:     workers,
		widtasks:    widtasks,
		workersList: ws,
	}, nil
}

// Execute returns a chan that receives the results generated by tasks execution.
// It calls the ExecuteEvents method and filters the returned results based on
// the Mode parameter.
func (eng *Engine) Execute(ctx context.Context, mode Mode) (chan Result, error) {

	// init the event chan
	eventchan, err := eng.ExecuteEvents(ctx)
	if err != nil {
		return nil, err
	}

	// func to filter the results to be exported
	type exportResultFn func(*Event) bool

	var exportResult exportResultFn

	switch mode {
	case FirstSuccessOrLastResult:
		exportResult = func(e *Event) bool { return e.IsFirstSuccessOrLastResult() }
	case ResultsUntilFirstSuccess:
		exportResult = func(e *Event) bool { return e.IsResultUntilFirstSuccess() }
	case SuccessOrErrorResults:
		exportResult = func(e *Event) bool { return e.IsSuccessOrError() }
	default:
		exportResult = func(e *Event) bool { return e.IsResult() }
	}

	// create the result chan
	resultchan := make(chan Result)

	// goroutine that read input from the event chan
	// write output to the result chan.
	go func(eventc chan *Event, resultc chan Result, export exportResultFn) {
		for e := range eventc {
			if export(e) {
				resultc <- e.Result
			}
		}
		close(resultc)
	}(eventchan, resultchan, exportResult)

	return resultchan, nil
}

// ExecuteEvents returns a chan that receives all the Events
// generated by each task execution.
// For each (worker, task) pair, it is emitted a Start event
// followed by a final event that can be Success, Canceled or Error.
//
// The method is useful to track the execution of the tasks:
// while the Execute method can only return the result on completion
// of execution, the ExecuteEvents method returns also the Start event
// at the beginning of execution (with a nil result).
func (eng *Engine) ExecuteEvents(ctx context.Context) (chan *Event, error) {

	if eng == nil {
		return nil, fmt.Errorf("nil engine")
	}
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}

	// creates the Event channel
	eventc := make(chan *Event)

	// creates the *jobOutput channel
	outputc := make(chan *jobOutput)

	// creates the *jobInput chan of each worker
	inputc := map[WorkerID](chan *jobInput){}
	for wid := range eng.workers {
		inputc[wid] = make(chan *jobInput)
	}

	// creates each task context
	taskctx := map[TaskID]context.Context{}
	taskcancel := map[TaskID]context.CancelFunc{}
	for _, ts := range eng.widtasks {
		for _, t := range ts {
			tid := t.TaskID()
			if _, ok := taskctx[tid]; !ok {
				ctx, cancel := context.WithCancel(ctx)
				taskctx[tid] = ctx
				taskcancel[tid] = cancel
			}
		}
	}

	// Starts the goroutines that executes the real work.
	// For each worker it starts N goroutines, with N = Instances.
	// Each goroutine get the input from the worker request channel,
	// and put the output to the task result channel (contained in the request).
	for _, worker := range eng.workersList {

		// for each worker instances
		for i := 0; i < worker.Instances; i++ {

			go func(w *Worker, inst int, inputc <-chan *jobInput) {
				for req := range inputc {

					timeStart := time.Now()

					// start event
					event := &Event{
						Task:       req.task,
						WorkerID:   w.WorkerID,
						WorkerInst: inst,
						Result:     nil,
						TaskStat:   req.stat,
						TimeStart:  timeStart,
						TimeEnd:    timeStart,
					}
					eventc <- event

					// get the worker result of the task
					res := w.Work(req.ctx, w, inst, req.task)

					// send the result to the output chan
					jout := jobOutput{
						wid:       w.WorkerID,
						instance:  inst,
						res:       res,
						task:      req.task,
						timeStart: timeStart,
						timeEnd:   time.Now(),
					}
					req.outc <- &jout
				}
			}(worker, i, inputc[worker.WorkerID])
		}
	}

	// start a goroutine that, for each worker instance,
	// send a void output to signal it is ready to work.
	go func() {
		for _, w := range eng.workersList {
			wid := w.WorkerID
			for i := 0; i < w.Instances; i++ {
				jout := jobOutput{
					wid:      wid,
					instance: i,
					res:      nil,
				}
				outputc <- &jout
			}
		}
	}()

	// main goroutine that handle the input and output from the workers
	// and send the events to the event chan.
	go func() {
		// clone eng.widtasks
		widtasks := eng.widtasks.Clone()

		// init the status map from the WorkerTasks object
		statMap := newTaskStatusMap(eng.widtasks)

		for !statMap.completed() {

			// get the next output
			o := <-outputc

			// handle result
			if o.res != nil {
				success := (o.res.Error() == nil)
				tid := o.task.TaskID()

				// updates task info map
				statMap.done(tid, success)

				if success {
					// call cancel func for the task context
					taskcancel[tid]()
				}

				// end event (success, error or canceled)
				event := &Event{
					Task:       o.task,
					WorkerID:   o.wid,
					WorkerInst: o.instance,
					Result:     o.res,
					TaskStat:   *statMap[tid],
					TimeStart:  o.timeStart,
					TimeEnd:    o.timeEnd,
				}
				eventc <- event
			}

			// select the next task of the worker
			var nexttask Task
			{
				ts := widtasks[o.wid]
				n := statMap.pick(ts)
				if n >= 0 {
					nexttask = ts.remove(n)
					widtasks[o.wid] = ts
				}
			}

			if nexttask == nil {
				// close the worker chan
				// NOTE: in case of a worker with two or more instances,
				// the close of the channel must be called only once.
				// Else get the error:
				//	  panic: close of closed channel
				if ch, ok := inputc[o.wid]; ok {
					close(ch)
					delete(inputc, o.wid)
				}

			} else {
				tid := nexttask.TaskID()

				// updates task info map
				statMap.doing(tid)

				// send the job to the worker
				i := &jobInput{
					ctx:    taskctx[tid],
					cancel: taskcancel[tid],
					task:   nexttask,
					outc:   outputc,
					stat:   *statMap[tid],
				}
				inputc[o.wid] <- i

			}
		}

		close(outputc)
		close(eventc)
	}()

	return eventc, nil
}

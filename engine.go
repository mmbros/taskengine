// Package taskengine can be used to concurrently execute a set of tasks
// assigned to multiple different workers.
//
// A Task represents a unit of work to be executed.
// Each task can be assigned to one or more workers.
// Two tasks are considered equivalent if they have the same TaskID.
// Note that tasks with the same TaskID can be different object with different information;
// this allows a task object assigned to a worker to contain information specific to that worker.
//
// Each Worker has a WorkFunc that performs the task.
// Multiple instances of the same worker can be used to concurrently execute
// different tasks assign to the worker.
//
// The execution mode of the task is managed by the engine.Mode parameters:
//
// - FirstSuccessOrLastError: For each task it returns only one result: the first success or the last error. If a task can be handled by two or more workers, only the first success result is returned. The remaining job for same task are skipped.
//
// - UntilFirstSuccess: For each task returns the (not successfull) result of all the workers: after the first success the other requests are cancelled.
//
// - All: For each task returns the result of all the workers. Multiple success results can be returned.
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
	// For each task returns only one result:
	// the first success or the last error.
	FirstSuccessOrLastError Mode = iota

	// For each task returns the results until the first success:
	// after the first success the other requests are cancelled and not returned.
	// At most one success is returned.
	UntilFirstSuccess

	// For each task returns the result of all the workers.
	// Multiple success results can be returned.
	// After the first success, the remaining requests are cancelled.
	All
)

// engine contains the workers and the tasks of each worker.
type engine struct {
	workers     map[WorkerID]*Worker
	widtasks    WorkerTasks // map[WorkerID]*Tasks
	ctx         context.Context
	workersList []*Worker // original workers list
}

// jobInput is the internal struct passed to a worker to execute a task.
type jobInput struct {
	// task context
	ctx context.Context

	// task cancel func
	cancel context.CancelFunc

	// task of the worker
	task Task

	// output channel
	outc chan *jobOutput

	// stat
	stat TaskStat
}

// jobOutput contains the result returned by the worker with the
// WorkerID and instance in executing the given task.
// A nil result indicates that the worker instance is ready to perform a task.
type jobOutput struct {
	wid            WorkerID
	instance       int
	res            Result // can be nil
	task           Task   // not used if res is nil
	Timestamp      time.Time
	TimestampStart time.Time
}

// newEngine initialize a new engine object from the list of workers and the tasks of each worker.
// It performs some sanity check and return error in case of incongruences.
func newEngine(ctx context.Context, ws []*Worker, wts WorkerTasks) (*engine, error) {

	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}

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

	return &engine{
		workers:     workers,
		widtasks:    widtasks,
		ctx:         ctx,
		workersList: ws,
	}, nil
}

// String representation of jobOutput object.
// Only for debug pourposes.
// TODO: do not compile in production code!
// func (o *jobOutput) String() string {
// 	var b strings.Builder

// 	b.WriteString("jobOutput{")
// 	if o == nil {
// 		b.WriteString("<nil>")
// 	} else {
// 		fmt.Fprintf(&b, "wid=%s, inst=%d, ", o.wid, o.instance)
// 		if o.res == nil {
// 			fmt.Fprint(&b, "res=<nil>")
// 		} else {
// 			fmt.Fprintf(&b, "tid=%s, success=%v", o.task.TaskID(), o.res.Success())
// 		}
// 	}
// 	b.WriteString("}")
// 	return b.String()
// }

// Execute returns a chan that receives the Results of the workers for the input Requests.
func (eng *engine) Execute(mode Mode) (chan Result, error) {

	if eng == nil {
		return nil, fmt.Errorf("nil engine")
	}

	// creates the Result channel
	resultc := make(chan Result)

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
				ctx, cancel := context.WithCancel(eng.ctx)
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
					// get the worker result of the task
					res := w.Work(req.ctx, inst, req.task)

					// send the result to the output chan
					jout := jobOutput{
						wid:      w.WorkerID,
						instance: inst,
						res:      res,
						task:     req.task,
					}
					req.outc <- &jout
				}
			}(worker, i, inputc[worker.WorkerID])
		}
	}

	// each worker instances send a void output
	// to signal it is ready to work.
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

	go func() {
		// clone eng.widtasks
		widtasks := eng.widtasks.Clone()

		// iter := 0
		statusMap := newTaskStatusMap(eng.widtasks)

		// for iter := 0; iter < totTasks; iter++ {
		for !statusMap.completed() {

			// iter++
			// if iter > 100 {
			// log.Println("MAX ITER !!!!")
			// break
			// }
			// log.Printf("iter: %02d\n", iter)
			// log.Println(tim)

			// get the next output
			o := <-outputc

			// log.Println(o)

			// handle result
			if o.res != nil {
				success := (o.res.Error() == nil)
				tid := o.task.TaskID()

				// updates task info map
				statusMap.done(tid, success)
				status := statusMap[tid]

				if success {
					// call cancel func for the task context
					taskcancel[tid]()
				}

				switch mode {
				case FirstSuccessOrLastError:
					if (success && status.Success == 1) || (status.Completed() && status.Success == 0) {
						// return the result if:
						// - it is the first success, or
						// - it is completed and no success was found
						resultc <- o.res
					}
				case UntilFirstSuccess:
					if (success && status.Success == 1) || (!success && status.Success == 0) {
						// return the result if:
						// - it is the first success, or
						// - it is a error and no success was found
						resultc <- o.res
					}
				default:
					resultc <- o.res
				}
			}

			// select the next task of the worker
			var nexttask Task
			{
				ts := widtasks[o.wid]
				n := statusMap.pick(ts)
				if n >= 0 {
					nexttask = ts.Remove(n)
					widtasks[o.wid] = ts
				}
			}

			if nexttask == nil {
				// log.Println("nexttask = <nil>")

				// close the worker chan
				// NOTE: in case of a worker with two or more instances,
				// the close of the channel must be called only once. Else
				//    panic: close of closed channel
				if ch, ok := inputc[o.wid]; ok {
					close(ch)
					delete(inputc, o.wid)
				}

			} else {
				tid := nexttask.TaskID()

				// updates task info map
				statusMap.doing(tid)

				// log.Printf("nexttask = %s\n", tid)

				i := &jobInput{
					ctx:    taskctx[tid],
					cancel: taskcancel[tid],
					task:   nexttask,
					outc:   outputc,
				}
				inputc[o.wid] <- i
			}
		}

		// log.Println("END LOOP")

		// log.Println(tim)

		close(outputc)
		close(resultc)
	}()

	return resultc, nil
}

// Execute returns a chan that receives the Results of the workers for the input Requests.
func (eng *engine) ExecuteEvent(mode Mode) (chan Event, error) {

	if eng == nil {
		return nil, fmt.Errorf("nil engine")
	}

	// creates the Event channel
	eventc := make(chan Event)

	// creates the Result channel
	// resultc := make(chan Result)

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
				ctx, cancel := context.WithCancel(eng.ctx)
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

					timestampStart := time.Now()

					// start event
					event := Event{
						Task:           req.task,
						Worker:         *w,
						Inst:           inst,
						Stat:           req.stat,
						Timestamp:      timestampStart,
						TimestampStart: timestampStart,
					}
					eventc <- event

					// get the worker result of the task
					res := w.Work(req.ctx, inst, req.task)

					// send the result to the output chan
					jout := jobOutput{
						wid:            w.WorkerID,
						instance:       inst,
						res:            res,
						task:           req.task,
						Timestamp:      time.Now(),
						TimestampStart: timestampStart,
					}
					req.outc <- &jout
				}
			}(worker, i, inputc[worker.WorkerID])
		}
	}

	// each worker instances send a void output
	// to signal it is ready to work.
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

	go func() {
		// clone eng.widtasks
		widtasks := eng.widtasks.Clone()

		// iter := 0
		statMap := newTaskStatusMap(eng.widtasks)

		// for iter := 0; iter < totTasks; iter++ {
		for !statMap.completed() {

			// iter++
			// if iter > 100 {
			// log.Println("MAX ITER !!!!")
			// break
			// }
			// log.Printf("iter: %02d\n", iter)
			// log.Println(tim)

			// get the next output
			o := <-outputc

			// log.Println(o)

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
				event := Event{
					Task:           o.task,
					Worker:         *eng.workers[o.wid],
					Inst:           o.instance,
					Result:         o.res,
					Stat:           *statMap[tid],
					Timestamp:      o.Timestamp,
					TimestampStart: o.TimestampStart,
				}
				eventc <- event

				// switch mode {
				// case FirstSuccessOrLastError:
				// 	if (success && status.success == 1) || (status.completed() && status.success == 0) {
				// 		// return the result if:
				// 		// - it is the first success, or
				// 		// - it is completed and no success was found
				// 		resultc <- o.res
				// 	}
				// case UntilFirstSuccess:
				// 	if (success && status.success == 1) || (!success && status.success == 0) {
				// 		// return the result if:
				// 		// - it is the first success, or
				// 		// - it is a error and no success was found
				// 		resultc <- o.res
				// 	}
				// default:
				// 	resultc <- o.res
				// }
			}

			// select the next task of the worker
			var nexttask Task
			{
				ts := widtasks[o.wid]
				n := statMap.pick(ts)
				if n >= 0 {
					nexttask = ts.Remove(n)
					widtasks[o.wid] = ts
				}
			}

			if nexttask == nil {
				// log.Println("nexttask = <nil>")

				// close the worker chan
				// NOTE: in case of a worker with two or more instances,
				// the close of the channel must be called only once. Else
				//    panic: close of closed channel
				if ch, ok := inputc[o.wid]; ok {
					close(ch)
					delete(inputc, o.wid)
				}

			} else {
				tid := nexttask.TaskID()

				// updates task info map
				statMap.doing(tid)

				// log.Printf("nexttask = %s\n", tid)

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

		// log.Println("END LOOP")

		// log.Println(tim)

		close(outputc)
		// close(resultc)
		close(eventc)
	}()

	return eventc, nil
}

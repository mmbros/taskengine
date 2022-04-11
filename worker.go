package taskengine

import (
	"context"
)

// Max number of instances for each worker
const maxInstances = 100

//-----------------------------------------------------------------------------
// Types to be customized if needed. For example:
//   type TaskID    int
//   type WorkerID  int

// WorkerID type definition.
type WorkerID string

// TaskID type definition.
type TaskID string

//-----------------------------------------------------------------------------

// Task is a unit of work that can be executed by a worker
// Two or more task with the same TaskID are equivalent
// and possibly only one will be executed.
// Two or more task with the same TaskID can contain different information
// usefull for a specific  worker.
type Task interface {
	TaskID() TaskID
}

// Result is the interface that must be matched by the output of the Work function.
type Result interface {
	// The error returned by the Work function.
	// It is used to determine the status of the task execution as follow
	//    Success:  error is nil
	//    Canceled: error is context.Canceled
	//    Error:    otherwise
	Error() error
}

// WorkFunc is the worker function to execute a given task.
// The int parameter represents the worker instance.
type WorkFunc func(context.Context, *Worker, int, Task) Result

// Worker is the unit (identified by WorkerID)
// that receives the Requests and
// executes a specific WorkFunc function to return the Responses.
// The Instances parameters represents the number of instances of each worker
type Worker struct {

	// Unique ID of the worker
	WorkerID WorkerID

	// Number of worker instances. Must be greater or equal 1
	Instances int

	// The work function
	Work WorkFunc
}

// Tasks is an array of tasks.
type Tasks []Task

// WorkerTasks is a map representing the tasks list of each worker
type WorkerTasks map[WorkerID]Tasks

// Clone method returns a cloned copy of the WorkerTasks object.
func (wts WorkerTasks) Clone() WorkerTasks {
	wts2 := WorkerTasks{}
	for w, ts := range wts {
		ts2 := Tasks{}
		ts2 = append(ts2, ts...)
		wts2[w] = ts2
	}
	return wts2
}

// remove removes the i-th task of the list.
// It returns the removed task.
// NOTE: DO NOT preserve the order of the items in the list.
// WARN: doen not check the i-th task exists!
func (ts *Tasks) remove(i int) Task {
	t := (*ts)[i]
	L1 := len(*ts) - 1
	(*ts)[i] = (*ts)[L1]
	(*ts) = (*ts)[:L1]
	return t
}

# taskengine

Package `taskengine` can be used to concurrently execute a set of tasks
assigned to multiple different workers.

Each worker can works all or a subset of the tasks.

When a worker is ready, the next task to work is dynamically choosen
considering the current status of the tasks
so to maximize the thoughput of the tasks successfully executed.

The main types defined by the package are:

- Engine
- Task
- Worker
- WorkerTasks

## Engine

The `NewEngine` function initialize a new `Engine` object given the list of
workers and the tasks of each worker.

    func NewEngine(ctx context.Context, ws []*Worker, wts WorkerTasks) (*Engine, error)

The `Execute` method of the engine object returns a chan in which are enqueued
the workers results for the input tasks.

    func (eng *Engine) Execute(mode Mode) (chan Result, error)

The `Mode` enum type represents the mode of execution:

- `AllResults`:
  for each task returns the result of all the workers:  
  success, error or canceled.
  After the first success (if exists) the remaining
  job for same task are cancelled and not returned.
  Multiple success results can be returned if they happen at the same time.

- `SuccessOrErrorResults`:
  for each task returns the success or error results.
  The canceled resuts are not returned.
  After the first success (if exists) the remaining
  job for same task are cancelled and not returned.
  Multiple success results can be returned if they happen at the same time.

- `ResultsUntilFirstSuccess`:
  for each task returns the results preceding the first success (included).
  After the first success (if exists) the remaining
  job for same task are cancelled and not returned.
  At most one success is returned.

- `FirstSuccessOrLastResult`:
  For each task returns only one result: the first success or the last result.
  After the first success (if exists) the remaining
  job for same task are cancelled and not returned.
  At most one success is returned.

## Task

A `Task` represents a unit of work to be executed. Each task can be
assigned to one or more workers.
Two tasks are considered equivalent if they have the same `TaskID`.  

**NOTE:** tasks with the same TaskID can be different object with different
information; this allows a task object assigned to a worker to contain
information specific to that worker.

    type Task interface {
        TaskID() TaskID      // Unique ID of the task
    }

## Worker

Each `Worker` has a `WorkFunc` that performs the task.
Multiple instances of the same worker can be used in order to execute
concurrently different tasks assign to the worker.  

    type Worker struct {
        WorkerID  WorkerID   // Unique ID of the worker
        Instances int        // Number of worker instances
        Work      WorkFunc   // The work function
    }

The `WorkFunc` receives in input a `context`, the `*Worker` and the instance
number of the worker and the `Task`, and returns an object that meets the
`Result` interface.

    type WorkFunc func(context.Context, *Worker, int, Task) Result

The `Result` interface has only the `Error` method.

    type Result interface {
        Error() error
    }

The returned error is used to determine the status of the task execution as follow:

- Success:  error is nil
- Canceled: error is context.Canceled
- Error:    otherwise

## WorkerTasks

`WorkerTasks` type is a map that contains the tasks list of each WorkerID.

    type WorkerTasks map[WorkerID]Tasks

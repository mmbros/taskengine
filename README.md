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

- `FirstSuccessOrLastError`: for each task it returns only one result:
  the first success or the last error. If a task can be handled by two
  or more workers, only the first success result is returned.
  The remaining job for same task are cancelled.

- `FirstSuccessThenCancel`: for each task it returns the error results
  preceding the first success and the first success. The remaining job
  for the same task are cancelled.

- `All`: for each task returns the result of all the workers.
  Multiple success results can be returned.

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

The `WorkFunc` receives in input a `context`, the instance number of the
worker and the `Task`, and returns an object that meets the `Result` interface.

    type WorkFunc func(context.Context, int, Task) Result

The `Result` interface has only the `Success` method that must returns true
in case of success and false otherwise.

    type Result interface {
        Success() bool
    }

## WorkerTasks

`WorkerTasks` type is a map that contains the tasks list of each WorkerID.

    type WorkerTasks map[WorkerID]Tasks

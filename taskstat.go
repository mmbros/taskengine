package taskengine

import "fmt"

// TaskStat type is used to dynamically choose the next task to work
type TaskStat struct {
	Todo    int // how many workers have to do the task.
	Doing   int // how many workers are doing the task.
	Done    int // how many workers have done the task.
	Success int // how many workers have done the task with success.
}

// func (stat *TaskStat) Todo() int    { return stat.todo }
// func (stat *TaskStat) Doing() int   { return stat.doing }
// func (stat *TaskStat) Done() int    { return stat.done }
// func (stat *TaskStat) Success() int { return stat.success }

// Completed returns if no worker has to do or is doing the task.
func (stat *TaskStat) Completed() bool {
	return (stat.Todo == 0) && (stat.Doing == 0)
}

// String representation of a TaskStat object.
func (stat TaskStat) String() string {
	return fmt.Sprintf("[%d %d %d(%d)]",
		stat.Todo, stat.Doing, stat.Done, stat.Success)
}

// taskStatMap maps TaskID -> taskInfo.
type taskStatMap map[TaskID]*TaskStat

// // total returns how many workers have assigned the task.
// func (stat *taskStat) total() int {
// 	return stat.todo + stat.doing + stat.done
// }

// // total returns how many workers have done the task with error.
// func (stat *taskStat) error() int {
// 	return stat.done - stat.success
// }

// newTaskStatusMap init a new taskInfoMap from a WorkerTasks object.
func newTaskStatusMap(widtasks WorkerTasks) taskStatMap {
	statmap := taskStatMap{}
	for _, ts := range widtasks {
		for _, t := range ts {
			statmap.todo(t.TaskID())
		}
	}
	return statmap
}

// completed returns true if all the tasks are completed:
// no worker has to do or is doing some task.
func (statmap taskStatMap) completed() bool {
	for _, stat := range statmap {
		if !stat.Completed() {
			return false
		}
	}
	return true
}

// todo increments the number workers that can perform the given task.
func (statmap taskStatMap) todo(tid TaskID) {
	stat := statmap[tid]
	if stat == nil {
		stat = &TaskStat{}
		statmap[tid] = stat
	}
	stat.Todo++
}

// doing increments the doing number and decrements the todo number.
// WARN: it doesn't check that task exists and todo > 0.
func (statmap taskStatMap) doing(tid TaskID) {
	stat := statmap[tid]
	stat.Todo--
	stat.Doing++
}

// done increments the done number and decrements the doing number.
// It also increments the success number, if needed.
// WARN: it doesn't check task exists and doing > 0.
func (statmap taskStatMap) done(tid TaskID, success bool) {
	stat := statmap[tid]
	stat.Doing--
	stat.Done++
	if success {
		stat.Success++
	}
}

// pick choose among the tasks list the best task to execute next.
// The task is chosen so to maximize the thoughput of the tasks successfully executed.
// It returns -1 if the tasks list is empty, or the index of the choosen task in the list.
// It doesn't updates neither the Tasks nor the taskInfoMap.
// WARN: it doesn't check every TaskID exists in taskStatMap.
func (statmap taskStatMap) pick(ts Tasks) int {
	L := len(ts)
	if L == 0 {
		return -1
	}

	j0 := 0
	s0 := statmap[ts[0].TaskID()]

	for j := 1; j < L; j++ {
		s := statmap[ts[j].TaskID()]

		if s.Success > s0.Success {
			// prefer task with fewer success
			continue
		} else if s.Success == s0.Success {
			if s.Doing > s0.Doing {
				// else prefer task with fewer doing
				continue
			} else if s.Doing == s0.Doing {
				if s.Todo > s0.Todo {
					// else prefer task with fewer todo
					continue
				} else if s.Todo == s0.Todo {
					// else prefer task with lower TaskID
					// NOTE: only needed to be deterministic
					tid0 := ts[j0].TaskID()
					tid := ts[j].TaskID()
					if tid >= tid0 {
						continue
					}
				}
			}
		}

		j0 = j
		s0 = s
	}

	return j0
}

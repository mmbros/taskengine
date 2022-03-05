package taskengine

// taskStat type is used to dynamically choose the next task to work
type taskStat struct {
	todo    int // how many workers have to do the task.
	doing   int // how many workers are doing the task.
	done    int // how many workers have done the task.
	success int // how many workers have done the task with success.
}

// taskStatMap maps TaskID -> taskInfo.
type taskStatMap map[TaskID]*taskStat

// // total returns how many workers have assigned the task.
// func (stat *taskStat) total() int {
// 	return stat.todo + stat.doing + stat.done
// }

// // total returns how many workers have done the task with error.
// func (stat *taskStat) error() int {
// 	return stat.done - stat.success
// }

// completed returns if no worker has to do or is doing the task.
func (stat *taskStat) completed() bool {
	return (stat.todo == 0) && (stat.doing == 0)
}

// newTaskStatusMap init a new taskInfoMap from WorkerTasks.
func newTaskStatusMap(widtasks WorkerTasks) taskStatMap {
	statmap := taskStatMap{}
	for _, ts := range widtasks {
		for _, t := range ts {
			statmap.todo(t.TaskID())
		}
	}
	return statmap
}

// completed returns if all the tasks are completed:
// no worker has to do or is doing some task.
func (statmap taskStatMap) completed() bool {
	for _, stat := range statmap {
		if !stat.completed() {
			return false
		}
	}
	return true
}

// todo increments the number workers that can perform the given task.
func (statmap taskStatMap) todo(tid TaskID) {
	stat := statmap[tid]
	if stat == nil {
		stat = &taskStat{}
		statmap[tid] = stat
	}
	stat.todo++
}

// doing decrements the todo number and increments the doing number.
// WARN: it doesn't check task exists and todo>0.
func (statmap taskStatMap) doing(tid TaskID) {
	stat := statmap[tid]
	stat.todo--
	stat.doing++
}

// done decrements the doing number and increments the done number.
// It also increments the success number, if needed.
// WARN: it doesn't check task exists and doing>0.
func (statmap taskStatMap) done(tid TaskID, success bool) {
	stat := statmap[tid]
	stat.doing--
	stat.done++
	if success {
		stat.success++
	}
}

// pick choose among the tasks list the best task to execute next.
// The task is chosen so to maximize the thoughput of the tasks successfully executed.
// It returns the index of the choosen task in the list.
// It doesn't updates the neither the Tasks nor the taskInfoMap.
// WARN: it doesn't check task exists.
func (statmap taskStatMap) pick(ts Tasks) int {
	L := len(ts)
	if L == 0 {
		return -1
	}

	j0 := 0
	s0 := statmap[ts[0].TaskID()]

	for j := 1; j < L; j++ {
		s := statmap[ts[j].TaskID()]

		if s.success > s0.success {
			// prefer task with fewer success
			continue
		} else if s.success == s0.success {
			if s.doing > s0.doing {
				// else prefer task with fewer doing
				continue
			} else if s.doing == s0.doing {
				if s.todo > s0.todo {
					// else prefer task with fewer todo
					continue
				} else if s.todo == s0.todo {
					// else prefer task with lower TaskID
					// needed to be deterministic
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

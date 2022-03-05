package taskengine

import (
	"fmt"
	"strings"
	"testing"
)

func (statmap taskStatMap) String() string {
	var b strings.Builder
	b.WriteString("[\n")
	for tid, ti := range statmap {
		fmt.Fprintf(&b, " %s: todo=%d, doing=%d, done=%d, success=%d\n",
			tid, ti.todo, ti.doing, ti.done, ti.success)
	}
	b.WriteString("]")
	return b.String()
}

type statTask string

func (t statTask) TaskID() TaskID { return TaskID(t) }

func TestPick(t *testing.T) {

	// TS tranforms a string "t1, t2, ..." to Tasks
	TS := func(s string) Tasks {
		a := strings.Split(s, ",")
		res := Tasks{}
		for _, tidx := range a {
			tid := strings.TrimSpace(tidx)
			if tid != "" {
				res = append(res, statTask(tid))
			}
		}
		return res
	}

	type testCase struct {
		statmap taskStatMap
		tasks   Tasks
		want    int
	}

	statmap123 := taskStatMap{
		"t1": &taskStat{1, 0, 0, 0},
		"t2": &taskStat{2, 0, 0, 0},
		"t3": &taskStat{3, 0, 0, 0},
	}

	testCases := map[string]testCase{
		"todo 123": {
			statmap: statmap123,
			tasks:   TS("t1,t2,t3"),
			want:    0,
		},
		"todo 32 alpha": {
			statmap: statmap123,
			tasks:   TS("t3,t2"),
			want:    1,
		},
		"todo 2 alpha": {
			statmap: statmap123,
			tasks:   TS("t2"),
			want:    0,
		},
		"todo -1": {
			statmap: statmap123,
			tasks:   TS(""),
			want:    -1,
		},
		// "123_y unknown task 0": {
		// 	statmap: statmap123,
		// 	tasks:   TS("t999"),
		// 	want:    0,
		// },
		// "123_y unknown task 1": {
		// 	statmap: statmap123,
		// 	tasks:   TS("t1, t999"),
		// 	want:    0,
		// },
		"done with success": {
			statmap: taskStatMap{
				"t1": &taskStat{1, 0, 1, 1},
				"t2": &taskStat{2, 0, 0, 0},
				"t3": &taskStat{3, 0, 0, 0},
			},
			tasks: TS("t1,t2,t3"),
			want:  1,
		},
		"done with error": {
			statmap: taskStatMap{
				"t1": &taskStat{1, 0, 1, 0},
				"t2": &taskStat{2, 0, 0, 0},
				"t3": &taskStat{3, 0, 0, 0},
			},
			tasks: TS("t1,t2,t3"),
			want:  0,
		},
		"doing": {
			statmap: taskStatMap{
				"t1": &taskStat{1, 1, 0, 0},
				"t2": &taskStat{2, 1, 0, 0},
				"t3": &taskStat{3, 0, 0, 0},
			},
			tasks: TS("t1,t2,t3"),
			want:  2,
		},
		"doing success alpha": {
			statmap: taskStatMap{
				"t1": &taskStat{1, 1, 0, 0},
				"t2": &taskStat{2, 1, 0, 0},
				"t3": &taskStat{3, 0, 1, 1},
			},
			tasks: TS("t2,t1,t3"),
			want:  1,
		},
	}

	for title, tc := range testCases {
		got := tc.statmap.pick(tc.tasks)

		if got != tc.want {
			t.Logf("%s: statmap = %v", title, tc.statmap)
			t.Logf("%s: tasks = %v", title, tc.tasks)
			t.Errorf("%s: want %d, got %d", title, tc.want, got)
		}
	}
}

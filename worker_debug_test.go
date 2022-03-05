// go:build debug

package taskengine

import (
	"strings"
	"testing"
)

func TestTasksString(t *testing.T) {

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
		input string
		want  string
	}

	testCases := map[string]testCase{
		"0 tasks": {
			input: "",
			want:  "[]",
		},
		"1 task": {
			input: "t1",
			want:  "[t1]",
		},
		"2 task": {
			input: "t1, t2",
			want:  "[t1, t2]",
		},
		"3 tasks": {
			input: "t1,t2,t3",
			want:  "[t1, t2, t3]",
		},
	}

	for title, tc := range testCases {
		ts := TS(tc.input)

		got := ts.String()

		if got != tc.want {
			t.Errorf("%s: want %q, got %q", title, tc.want, got)
		}
	}

}

func TestWorkerTasksString(t *testing.T) {

	type testCase struct {
		input map[string]testCaseTasks
		want  string
	}

	testCases := map[string]testCase{
		"empty": {
			input: map[string]testCaseTasks{},
			want: `{
}
`,
		},
		"aaa": {
			input: map[string]testCaseTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, true}},
				"w2": {{"t3", 10, true}},
				"w3": {{"t3", 20, true}, {"t2", 10, true}},
			},
			want: `{
   w1 : [t3, t2, t1]
   w2 : [t3]
   w3 : [t3, t2]
}
`,
		},
	}

	for title, tc := range testCases {
		tasks := newTestWorkeridTasks(t, tc.input)
		got := tasks.String()

		if got != tc.want {
			t.Errorf("%s: want %q, got %q", title, tc.want, got)
		}
	}

}

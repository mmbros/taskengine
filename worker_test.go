package taskengine

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWorkerTasks_Clone(t *testing.T) {

	tests := []struct {
		name  string
		input map[string]testingTasks
	}{
		{
			name:  "empty",
			input: map[string]testingTasks{},
		},
		{
			name: "not empty",
			input: map[string]testingTasks{
				"w1": {{"t1", 11, true}, {"t2", 12, true}, {"t3", 13, false}},
				"w2": {{"t1", 21, false}, {"t2", 22, true}},
				"w3": {{"t1", 31, true}},
			},
		},
	}

	copts := cmp.Options{cmp.Comparer(comparerTestingTask)}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := testingWorkerTasks(tt.input)
			got := want.Clone()
			if diff := cmp.Diff(want, got, copts); diff != "" {
				t.Errorf("Tasks.Clone() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTasks_remove(t *testing.T) {

	t1 := testingTask{"t1", 11, true}
	t2 := testingTask{"t2", 12, true}
	t3 := testingTask{"t3", 13, false}
	t4 := testingTask{"t4", 14, false}

	tests := []struct {
		name      string
		input     testingTasks
		idx       int
		wantTask  testingTask
		wantTasks testingTasks
	}{
		{
			name:      "remove first",
			input:     testingTasks{t1, t2, t3, t4},
			idx:       0,
			wantTask:  t1,
			wantTasks: testingTasks{t4, t2, t3},
		},
		{
			name:      "remove second",
			input:     testingTasks{t1, t2, t3, t4},
			idx:       1,
			wantTask:  t2,
			wantTasks: testingTasks{t1, t4, t3},
		},
		{
			name:      "remove third",
			input:     testingTasks{t1, t2, t3, t4},
			idx:       2,
			wantTask:  t3,
			wantTasks: testingTasks{t1, t2, t4},
		},
		{
			name:      "remove last",
			input:     testingTasks{t1, t2, t3, t4},
			idx:       3,
			wantTask:  t4,
			wantTasks: testingTasks{t1, t2, t3},
		},
	}

	copts := cmp.Options{cmp.Comparer(comparerTestingTask)}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := testingTasksToTasks(tt.input)

			task := tasks.remove(tt.idx)

			got := task.(testingTask)
			wantTasks := testingTasksToTasks(tt.wantTasks)

			if diff := cmp.Diff(tt.wantTask, got, copts); diff != "" {
				t.Errorf("Tasks.remove() mismatch task (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(wantTasks, tasks, copts); diff != "" {
				t.Errorf("Tasks.remove() mismatch tasks (-want +got):\n%s", diff)
			}
		})
	}
}
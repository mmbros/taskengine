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

func TestGolang_ListOfPointer(t *testing.T) {

	t1 := &testingTask{"t1", 11, true}
	t2 := &testingTask{"t2", 12, true}

	tasks := []*testingTask{t1, t2}

	if t1 != tasks[0] {
		t.Errorf("Pointer: tasks[0]:%p  != t1:%p", tasks[0], t1)
	}
}

func TestGolang_ListOfStruct(t *testing.T) {
	t1 := testingTask{"t1", 11, true}
	t2 := testingTask{"t2", 12, true}

	// the item of the list are copied
	tasks := []testingTask{t1, t2}

	// if you change the list item, the original item remain unchanged
	tasks[0].taskid = "new"

	want := "t1"
	if t1.taskid != want {
		t.Errorf("t1.taskid: want %q, got %s", want, t1.taskid)
	}

	want = "new"
	if tasks[0].taskid != want {
		t.Errorf("tasks[0].taskid: want %q, got %s", want, tasks[0].taskid)
	}
}

func TestGolang_ListOfInterface(t *testing.T) {
	t1 := Task(testingTask{"t1", 11, true})
	t2 := Task(testingTask{"t2", 12, true})

	// the item of the list are copied
	tasks := []Task{t1, t2}

	// if you change the list item, the original item remain unchanged
	task0 := tasks[0].(testingTask)

	task0.taskid = "new"

	want := "t1"
	if t1.(testingTask).taskid != want {
		t.Errorf("t1.taskid: want %q, got %s", want, t1.(testingTask).taskid)
	}

	want = "new"
	if task0.taskid != want {
		t.Errorf("task0.taskid: want %q, got %s", want, task0.taskid)
	}

	want = "t1"
	got := tasks[0].(testingTask).taskid
	if got != want {
		t.Errorf("tasks[0].(testingTask).taskid: want %q, got %s", want, got)
	}
}

func TestGolang_ListOfInterfacePointer(t *testing.T) {
	t1 := Task(&testingTask{"t1", 11, true})
	t2 := Task(&testingTask{"t2", 12, true})

	// the item of the list are copied
	tasks := []Task{t1, t2}

	// if you change the list item, the original item remain change
	task0 := tasks[0].(*testingTask)

	task0.taskid = "new"

	want := "new"
	if t1.(*testingTask).taskid != want {
		t.Errorf("t1.taskid: want %q, got %s", want, t1.(*testingTask).taskid)
	}

	want = "new"
	if task0.taskid != want {
		t.Errorf("task0.taskid: want %q, got %s", want, task0.taskid)
	}

	want = "new"
	got := tasks[0].(*testingTask).taskid
	if got != want {
		t.Errorf("tasks[0].(testingTask).taskid: want %q, got %s", want, got)
	}
}

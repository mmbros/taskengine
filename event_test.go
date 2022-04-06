package taskengine

import (
	"context"
	"testing"
)

func TestEvents(t *testing.T) {

	tests := []struct {
		name    string
		workers []*Worker
		input   map[string]testingTasks
		want    []testingEventsGroup
	}{
		{
			name: "one worker",
			workers: []*Worker{
				{"w1", 1, testingWorkFn},
			},
			input: map[string]testingTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, true}, {"t1", 10, false}},
			},
			want: []testingEventsGroup{
				{{"w1", "t1", EventStart}},
				{{"w1", "t1", EventError}},
				{{"w1", "t2", EventStart}},
				{{"w1", "t2", EventSuccess}},
				{{"w1", "t3", EventStart}},
				{{"w1", "t3", EventSuccess}},
			},
		},
		{
			name: "one worker two instances",
			workers: []*Worker{
				{"w1", 2, testingWorkFn},
			},
			input: map[string]testingTasks{
				"w1": {{"t3", 30, true}, {"t2", 20, false}, {"t1", 10, false}},
			},
			want: []testingEventsGroup{
				{{"w1", "t1", EventStart}, {"w1", "t2", EventStart}},
				{{"w1", "t1", EventError}, {"w1", "t2", EventError}, {"w1", "t3", EventStart}},
				{{"w1", "t3", EventSuccess}},
			},
		},
		{
			name: "three workers same task",
			workers: []*Worker{
				{"w1", 1, testingWorkFn},
				{"w2", 2, testingWorkFn},
				{"w3", 3, testingWorkFn},
			},
			input: map[string]testingTasks{
				"w1": {{"t1", 10, false}},
				"w2": {{"t1", 20, true}},
				"w3": {{"t1", 30, true}},
			},
			want: []testingEventsGroup{
				{{"w1", "t1", EventStart}, {"w2", "t1", EventStart}, {"w3", "t1", EventStart}},
				{{"w1", "t1", EventError}},
				{{"w2", "t1", EventSuccess}},
				{{"w3", "t1", EventCanceled}},
			},
		},

		{
			name: "two workers",
			workers: []*Worker{
				{"w1", 1, testingWorkFn},
				{"w2", 1, testingWorkFn},
			},
			input: map[string]testingTasks{
				"w1": {{"t3", 10, true}, {"t2", 10, true}, {"t1", 10, false}},
				"w2": {{"t3", 10, true}, {"t2", 5, false}},
			},
			want: []testingEventsGroup{
				{{"w1", "t1", EventStart}, {"w2", "t2", EventStart}},
				{{"w1", "t1", EventError}, {"w2", "t2", EventError}, {"w1", "t2", EventStart}, {"w2", "t3", EventStart}},
				{{"w2", "t3", EventSuccess}},
				{{"w1", "t2", EventSuccess}},
				{{"w1", "t3", EventStart}},
				{{"w1", "t3", EventCanceled}},
			},
		},
		{
			name: "three workers",
			workers: []*Worker{
				{"w1", 1, testingWorkFn},
				{"w2", 1, testingWorkFn},
				{"w3", 1, testingWorkFn},
			},
			input: map[string]testingTasks{
				"w1": {{"t3", 30, false}, {"t2", 20, true}, {"t1", 6, true}},
				"w2": {{"t3", 20, false}, {"t2", 8, true}},
				"w3": {{"t3", 10, false}},
			},
			want: []testingEventsGroup{
				{{"w1", "t1", EventStart}, {"w2", "t2", EventStart}, {"w3", "t3", EventStart}},
				{{"w1", "t1", EventSuccess}},
				{{"w1", "t2", EventStart}},
				{{"w2", "t2", EventSuccess}},
				{{"w2", "t3", EventStart}, {"w1", "t2", EventCanceled}},
				{{"w1", "t3", EventStart}},
				{{"w3", "t3", EventError}},
				{{"w2", "t3", EventError}},
				{{"w1", "t3", EventError}},
			},
		},
	}

	// copts := cmp.Options{}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wts := testingWorkerTasks(tt.input)
			eng, err := newEngine(ctx, tt.workers, wts)

			if err != nil {
				t.Errorf("newEngine: unexpected error: %s", err)
			}

			eventc, err := eng.ExecuteEvent(All)
			if err != nil {
				t.Errorf("ExecuteEvent: unexpected error: %s", err)
			}

			got := []Event{}
			for ev := range eventc {
				t.Log(ev)
				got = append(got, ev)
			}

			if diff := testingEventsDiff(tt.want, got); diff != "" {
				t.Errorf("%s: mismatch (-want +got):\n%s", tt.name, diff)
			}
		})
	}
}

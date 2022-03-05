// go:build debug

package taskengine

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// String representation of a Tasks object.
func (ts Tasks) String() string {
	a := make([]string, 0, len(ts))
	for _, t := range ts {
		a = append(a, string(t.TaskID()))
	}
	return "[" + strings.Join(a, ", ") + "]"
}

// String representation of a WorkerTasks object.
func (wts WorkerTasks) String() string {

	// build the array of (the string representation of) workerID sorted alphabetically
	wids := make([]string, 0, len(wts))
	for wid := range wts {
		wids = append(wids, string(wid))
	}
	sort.Strings(wids)

	// print to a buffer
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "{\n")
	for _, wid := range wids {
		fmt.Fprintf(buf, "   %s : %s\n", wid, wts[WorkerID(wid)])
	}
	fmt.Fprintf(buf, "}\n")

	// buffer to string
	return buf.String()
}

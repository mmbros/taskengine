package taskengine

import (
	"context"
	"testing"
)

func TestExecuteFunc(t *testing.T) {
	var ctx context.Context
	_, err := NewEngine(ctx, nil, nil)
	if err == nil {
		t.Errorf("Expecting error, got no error")
	}
}

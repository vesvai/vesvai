package agent

import (
	"context"
	"testing"
)

func TestWithAgent_FromContext(t *testing.T) {
	a := New("ctx-agent")
	ctx := WithAgent(context.Background(), a)
	if got := FromContext(ctx); got != a {
		t.Errorf("FromContext = %p, want %p", got, a)
	}
}

func TestFromContext_NoAgent(t *testing.T) {
	if got := FromContext(context.Background()); got != nil {
		t.Errorf("FromContext = %v, want nil", got)
	}
}

func TestWithAgent_Nil(t *testing.T) {
	ctx := WithAgent(context.Background(), nil)
	if got := FromContext(ctx); got != nil {
		t.Errorf("FromContext = %v, want nil", got)
	}
}

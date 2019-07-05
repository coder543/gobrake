package gobrake

import (
	"context"
)

type QueueTrace struct {
	trace
	Queue   string
	Errored bool
}

var _ Trace = (*QueueTrace)(nil)

func NewQueueTrace(c context.Context, name string) (context.Context, *QueueTrace) {
	t := &QueueTrace{
		Queue: name,
	}
	t.startTime = clock.Now()
	if c != nil {
		c = withTrace(c, t)
	}
	return c, t
}

func ContextQueueTrace(c context.Context) *QueueTrace {
	if c == nil {
		return nil
	}
	t, _ := c.Value(traceCtxKey).(*QueueTrace)
	return t
}

func (t *QueueTrace) StartSpan(name string) Span {
	if t == nil {
		return noopSpan{}
	}
	return t.trace.StartSpan(name)
}

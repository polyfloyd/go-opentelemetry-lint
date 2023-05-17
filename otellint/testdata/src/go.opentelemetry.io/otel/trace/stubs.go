package trace

import "context"

type Tracer interface {
	Start(context.Context, string) (context.Context, Span)
}

type Span interface {
	End()
}

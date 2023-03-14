package lint

import (
	"context"
	"fmt"
	"net/http"
)

const tracerName = "test"

func MissingSpan(ctx context.Context) { // want "Missing OpenTelemetry span"
	fmt.Println("hi")
}

func HTTPHanderMissingSpan(w http.ResponseWriter, r *http.Request) { // want "Missing OpenTelemetry span"
	fmt.Fprintf(w, "hi")
}

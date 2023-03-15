package lint

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
)

const tracerName = "test"

func SpanOk(ctx context.Context, db *sql.DB) error {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "SpanOk")
	defer span.End()

	row := db.QueryRowContext(ctx, `SELECT * FROM sample_text`)
	return row.Err()
}

func MissingSpan(ctx context.Context) { // want "Missing OpenTelemetry span"
	fmt.Println("hi")
}

func HTTPHanderMissingSpan(w http.ResponseWriter, r *http.Request) { // want "Missing OpenTelemetry span"
	fmt.Fprintf(w, "hi")
}

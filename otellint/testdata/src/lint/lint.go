package lint

import (
	"context"
	"database/sql"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "testdata/lint"

func tracer() trace.Tracer {
	return otel.Tracer("testdata/lint")
}

func AddToContext(ctx context.Context, thing string) context.Context {
	return context.WithValue(ctx, 1337, thing)
}

func AddToContextButCanFail(ctx context.Context, thing string) (context.Context, error) {
	return context.WithValue(ctx, 1337, thing), nil
}

func GetFromContext(ctx context.Context) string {
	return ctx.Value(1337).(string)
}

func SpanOk(ctx context.Context, db *sql.DB) error {
	ctx, span := tracer().Start(ctx, "SpanOk")
	defer span.End()

	row := db.QueryRowContext(ctx, `SELECT * FROM sample_text`)
	return row.Err()
}

func SpanOkOtherStyle(ctx context.Context, db *sql.DB) error {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "SpanOkOtherStyle")
	defer span.End()

	row := db.QueryRowContext(ctx, `SELECT * FROM sample_text`)
	return row.Err()
}

func SpanMisspelled(ctx context.Context, db *sql.DB) error {
	ctx, span := tracer().Start(ctx, "queryThing") // want "OpenTelemetry span misspelled, expected `SpanMisspelled`"
	defer span.End()

	row := db.QueryRowContext(ctx, `SELECT * FROM sample_text`)
	return row.Err()
}

type querier struct {
	db *sql.DB
}

func (q *querier) Query(ctx context.Context) error { // want "Missing OpenTelemetry span"
	row := q.db.QueryRowContext(ctx, `SELECT * FROM sample_text`)
	return row.Err()
}

func (q *querier) Query2(ctx context.Context) error {
	ctx, span := tracer().Start(ctx, "queryThing") // want "OpenTelemetry span misspelled, expected `\\(\\*querier\\).Query2`"
	defer span.End()

	row := q.db.QueryRowContext(ctx, `SELECT * FROM sample_text`)
	return row.Err()
}

func (q *querier) Query3(c context.Context) error { // want "Missing OpenTelemetry span"
	row := q.db.QueryRowContext(c, `SELECT * FROM sample_text`)
	return row.Err()
}

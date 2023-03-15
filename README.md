go-opentelemetry-lint
=====================

[![Build Status](https://github.com/polyfloyd/go-opentelemetry-lint/workflows/CI/badge.svg)](https://github.com/polyfloyd/go-opentelemetry-lint/actions)

This linter helps with properly using the OpenTelemetry SDK for Go.

## Usage
go-opentelemetry-lint accepts a set of package names similar to golint:
```
go-errorlint ./...
```

The linter is disabled for:
* Files containing unit tests
* Files that have `DO NOT EDIT` in their top level comment, as is typical for automatically
  generated files

## Lints

### Missing Spans
```
// bad
func Query(ctx context.Context, db *sql.DB) error {
	row := db.QueryRowContext(ctx, `SELECT * FROM sample_text`)
	return row.Err()
}

// good
func Query(ctx context.Context, db *sql.DB) error {
	ctx, span := tracer().Start(ctx, "query")
	defer span.End()

	row := db.QueryRowContext(ctx, `SELECT * FROM sample_text`)
	return row.Err()
}
```

The linter is able to find functions that take a context parameter but do not have a span. When
`-fix` is set, the missing spans will appear like magic.

Functions that implement `http.HandlerFunc` are also checked. For such functions, a span is created
from the HTTP request's context.

Functions that are deemed to be context "meta" functions (such as getting and setting variables)
excluded. The linter uses these heuristics to identify these cases:
* Functions that include a `context.Context` in their return
* Functions that only ever call `(Context).Value()` and do not use the context in other ways

A span is started by calling `Start` on an `otel.Tracer` instance. The linter has two ways of
writing the expression to get the Tracer, `Func` (the default) and `Const` which are the respective
arguments to the `-tracer-style` flag:
* `Func`: Via a function that returns it: `tracer().Start(..)`
* `Const`: Calling otel.Tracer with a const defined in the package: `otel.Start(tracerName).Start(..)`

Caveats / TODOs:
* When a span is added and a context parameter is not used in the body, compilation will fail. It is
  recommended to add a `_ = ctx` line manually in such cases
* When a span added to a HTTP handler function, other calls to `(*http.Request).Context` must be
  changed to point to the new span context
* When a span is added to a function which does have a `context.Context` as first argument but this
  argument is not named, the name is defaulted to `ctx` and requires this context to be named
  manually
* If spans are newly introduced to a file, OpenTelemetry imports are not added.

### Misspelled span names
A lint exists to flag cases where a span is started but with a name that does not equal the
function. An automatic fix is available which will alter only the name, but no other aspects of the
span invocation.

For separate functions, the expected span name is the name of the function. For member methods of a
struct, the span name is formatted as the method expression syntax of the function, e.g.
`(*fooService).GetFoo`.

## Contributing

Do you think you have found a bug? Then please report it via the Github issue tracker. Make sure to
attach any problematic files that can be used to reproduce the issue. Such files are also used to
create regression tests that ensure that your bug will never return.

When submitting pull requests, please prefix your commit messages with `fix:` or `feat:` for bug
fixes and new features respectively. This is the
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) scheme that is used to
automate some maintenance chores such as generating the changelog and inferring the next version
number.

go-opentelemetry-lint
=====================

[![Build Status](https://github.com/polyfloyd/go-opentelemetry-lint/workflows/CI/badge.svg)](https://github.com/polyfloyd/go-opentelemetry-lint/actions)

This linter helps with properly using the OpenTelemetry SDK for Go.

Implemented functionality:
* Find spans that with names that do not match with the function they are contained in (+automatic fix)
* Find functions that do not have spans at all (+automatic fix)

WIP!

- [x] misspelled spans + autofix
- [x] missing spans + autofix
- [x] ignore context returning functions
- [x] ignore DO NOT EDIT files
- [x] ignore context.Value functions
- [x] http handler contexts
- [ ] provide `_ = ctx` if ctx is not used in the body itself
- [ ] flag to configure tracer() or tracerName pattern
- [ ] missing span.End
- [ ] span.End not deferred
- [ ] replace http r.Context() with ctx if a span was added

## Contributing

Do you think you have found a bug? Then please report it via the Github issue tracker. Make sure to
attach any problematic files that can be used to reproduce the issue. Such files are also used to
create regression tests that ensure that your bug will never return.

When submitting pull requests, please prefix your commit messages with `fix:` or `feat:` for bug
fixes and new features respectively. This is the
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) scheme that is used to
automate some maintenance chores such as generating the changelog and inferring the next version
number.

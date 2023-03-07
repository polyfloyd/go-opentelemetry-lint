package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/polyfloyd/go-opentelemetry-lint/otellint"
)

func main() {
	singlechecker.Main(otellint.NewAnalyzer())
}

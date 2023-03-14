package otellint

import (
	"log"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestLints(t *testing.T) {
	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), NewAnalyzer(), "lint")
}

func TestTracerStyleConst(t *testing.T) {
	analyzer := NewAnalyzer()
	if err := analyzer.Flags.Set("tracer-style", "const"); err != nil {
		log.Fatal(err)
	}
	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzer, "tracer-style-const")
}

package otellint

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestLints(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), NewAnalyzer(), "lint")
}

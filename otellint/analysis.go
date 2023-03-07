package otellint

import (
	"flag"
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

func NewAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:  "opentelemetry",
		Doc:   "Go linter for services that use OpenTelemetry instrumentation. It can locate and fix missing spans and spans which' name does not match the function they are in",
		Run:   run,
		Flags: flagSet,
	}
}

var (
	flagSet flag.FlagSet
)

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			lintFunction(pass, fn)
		}
	}
	return nil, nil
}

func lintFunction(pass *analysis.Pass, fn *ast.FuncDecl) {
	field := findFunctionContextArgument(pass, fn)
	if field == nil {
		return
	}
	if isFunctionReturningContext(pass, fn) {
		return
	}

	funcName := fullFuncName(pass, fn)

	otelStartCall, indexInStmtList := findFunctionOtelSpan(pass, fn)
	if otelStartCall == nil {
		pass.Report(analysis.Diagnostic{
			Pos:     fn.Type.Pos(),
			Message: fmt.Sprintf("Missing OpenTelemetry span for `%s`", funcName),
			// TODO: SuggestedFixes:
		})
		return
	}

	spanNameArg, ok := otelStartCall.Args[1].(*ast.BasicLit)
	if !ok {
		return
	}
	spanName := spanNameArg.Value[1 : len(spanNameArg.Value)-1]

	if spanName != funcName {
		pass.Report(analysis.Diagnostic{
			Pos:     spanNameArg.ValuePos,
			Message: fmt.Sprintf("OpenTelemetry span misspelled, expected `%s`", funcName),
			// TODO: SuggestedFixes:
		})
	}

	_ = indexInStmtList
}

func findFunctionContextArgument(pass *analysis.Pass, fn *ast.FuncDecl) *ast.Ident {
	if len(fn.Type.Params.List) == 0 {
		return nil // Ignore functions without parameters
	}
	field := fn.Type.Params.List[0]
	if len(field.Names) != 1 {
		return nil // Ignore multiple context arguments.
	}

	typ := pass.TypesInfo.Types[field.Type].Type
	if typ.String() != "context.Context" {
		return nil
	}

	return field.Names[0]
}

func isFunctionReturningContext(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil {
		return false
	}
	for _, field := range fn.Type.Results.List {
		typ := pass.TypesInfo.Types[field.Type].Type
		if typ.String() == "context.Context" {
			return true
		}
	}
	return false
}

func findFunctionOtelSpan(pass *analysis.Pass, fn *ast.FuncDecl) (*ast.CallExpr, int) {
	for indexInStmtList, stmt := range fn.Body.List {
		assign, ok := stmt.(*ast.AssignStmt)
		if !ok {
			continue
		}

		if len(assign.Rhs) != 1 {
			continue
		}

		rhs := assign.Rhs[0]
		typ := pass.TypesInfo.Types[rhs].Type

		tup, ok := typ.(*types.Tuple)
		if !ok {
			continue
		}
		if tup.String() != "(context.Context, go.opentelemetry.io/otel/trace.Span)" {
			continue
		}
		otelStartCall, ok := rhs.(*ast.CallExpr)
		if !ok {
			continue
		}
		if otelStartCall.Fun.(*ast.SelectorExpr).Sel.Name != "Start" {
			continue
		}

		return otelStartCall, indexInStmtList
	}
	return nil, -1
}

func fullFuncName(pass *analysis.Pass, fn *ast.FuncDecl) string {
	if fn.Recv == nil {
		return fn.Name.Name
	}
	recv0 := fn.Recv.List[0]
	recvTyp := pass.TypesInfo.Types[recv0.Type].Type.String()
	split := strings.Split(recvTyp, ".")
	recvTypWithoutPackage := split[len(split)-1]
	ptr := ""
	if recvTyp[0] == '*' {
		ptr = "*"
	}
	return fmt.Sprintf("(%s%s).%s", ptr, recvTypWithoutPackage, fn.Name.Name)
}

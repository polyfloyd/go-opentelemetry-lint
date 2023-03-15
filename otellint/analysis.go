package otellint

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
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
	flagSet     flag.FlagSet
	tracerStyle TracerStyle
	tracerName  string
)

func init() {
	flagSet.StringVar((*string)(&tracerStyle), "tracer-style", string(TracerStyleFunc), "How the otel.Tracer should be invoked")
	flagSet.StringVar(&tracerName, "tracer-name", "tracer", "The name of the function or const that should be invoced to get an otel.Tracer")
}

type TracerStyle string

const (
	TracerStyleFunc  TracerStyle = "func"
	TracerStyleConst TracerStyle = "const"
)

func run(pass *analysis.Pass) (interface{}, error) {
	switch tracerStyle {
	case TracerStyleFunc, TracerStyleConst:
	default:
		return nil, fmt.Errorf("invalid tracer-style: %q", tracerStyle)
	}
	if tracerName == "" {
		return nil, fmt.Errorf("tracer-name is empty")
	}

	for _, file := range pass.Files {
		if isFileDoNotEdit(file) {
			continue
		}
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

func isFileDoNotEdit(file *ast.File) bool {
	for _, group := range file.Comments {
		for _, c := range group.List {
			return strings.Contains(c.Text, "DO NOT EDIT")
		}
	}
	return false
}

func lintFunction(pass *analysis.Pass, fn *ast.FuncDecl) {
	contextExpr := findFunctionContextArgument(pass, fn)
	if contextExpr == nil {
		return
	}

	// Functions that acts as wrappers for setting and getting values set on
	// the context are exempt.
	if isFunctionReturningContext(pass, fn) {
		return
	}
	if ident, ok := contextExpr.(*ast.Ident); ok {
		if isFunctionUsingContextValueOnly(pass, fn, ident.Obj) {
			return
		}
	}

	funcName := fullFuncName(pass, fn)

	otelStartCall, indexInStmtList := findFunctionOtelSpan(pass, fn)
	if otelStartCall == nil {
		var contextVar *ast.Ident
		if ident, ok := contextExpr.(*ast.Ident); ok {
			contextVar = ident
		} else {
			contextVar = &ast.Ident{NamePos: token.NoPos, Name: "ctx", Obj: nil}
			// TODO: Look for r.Context() instances and replace those with `ctx`.
		}

		insertPos := fn.Body.Rbrace
		if len(fn.Body.List) > 0 {
			insertPos = fn.Body.List[0].Pos()
		}
		pass.Report(analysis.Diagnostic{
			Pos:     fn.Type.Pos(),
			Message: fmt.Sprintf("Missing OpenTelemetry span for `%s`", funcName),
			SuggestedFixes: []analysis.SuggestedFix{{
				Message: "Insert span",
				TextEdits: []analysis.TextEdit{{
					Pos:     insertPos,
					End:     insertPos,
					NewText: spanCallSrc(pass, contextExpr, contextVar, funcName),
				}},
			}},
		})
		return
	}

	spanNameLit := otelStartCall.Args[1]
	spanNameArg, ok := spanNameLit.(*ast.BasicLit)
	if !ok {
		return
	}
	spanName := spanNameArg.Value[1 : len(spanNameArg.Value)-1]

	if spanName != funcName {
		pass.Report(analysis.Diagnostic{
			Pos:     spanNameArg.ValuePos,
			Message: fmt.Sprintf("OpenTelemetry span misspelled, expected `%s`", funcName),
			SuggestedFixes: []analysis.SuggestedFix{{
				Message: "Alter span name",
				TextEdits: []analysis.TextEdit{{
					Pos:     spanNameLit.Pos(),
					End:     spanNameLit.End(),
					NewText: []byte(fmt.Sprintf("%q", funcName)),
				}},
			}},
		})
	}

	_ = indexInStmtList
}

// findFunctionContextArgument returns the expression that can be used to
// obtain the context passed.
func findFunctionContextArgument(pass *analysis.Pass, fn *ast.FuncDecl) ast.Expr {
	sig, ok := pass.TypesInfo.ObjectOf(fn.Name).Type().(*types.Signature)
	if !ok {
		return nil
	}

	// If the function takes a context.Context as first argument, the
	// corresponding identifier is returned.
	if sig.Params().Len() >= 1 {
		argType0 := sig.Params().At(0).Type().String()
		if argType0 == "context.Context" {
			argField := fn.Type.Params.List[0]
			if len(argField.Names) == 0 {
				return mustParseExpr("ctx")
			}
			return argField.Names[0]
		}
	}

	// If the function arguments are compatible with net/http.HandlerFunc, then
	// an expression that describes http.Request.Context() is returned.
	if sig.Params().Len() == 2 {
		argType0 := sig.Params().At(0).Type().String()
		argType1 := sig.Params().At(1).Type().String()
		if argType0 == "net/http.ResponseWriter" && argType1 == "*net/http.Request" {
			r := fn.Type.Params.List[1].Names[0]
			return mustParseExpr(fmt.Sprintf("%s.Context()", r))
		}
	}

	return nil
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

func isFunctionUsingContextValueOnly(pass *analysis.Pass, fn *ast.FuncDecl, contextObj *ast.Object) bool {
	// Check whether the context variable is referenced in, and only in,
	// context.Value function calls.
	//
	// We do this by:
	// * Counting how many times the context is referenced in the whole function body
	// * Counting how many times context.Value is called
	//
	// A context.Value call counts as one general reference. Which means that
	// if both counts are equal, only context.Value is called.

	referenceCount := 0
	referenceCountIgnore := 0

	ast.Walk(astVisitorFunc(func(node ast.Node) {
		if ident, ok := node.(*ast.Ident); ok && ident.Obj == contextObj {
			referenceCount++
		}

		call, ok := node.(*ast.CallExpr)
		if !ok {
			return
		}
		fnSel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		if ident, ok := fnSel.X.(*ast.Ident); !ok || ident.Obj != contextObj {
			return
		}
		if fnSel.Sel.Name != "Value" {
			return
		}

		referenceCountIgnore++
	}), fn.Body)

	if referenceCountIgnore > 0 {
		return referenceCount == referenceCountIgnore
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

func spanCallSrc(pass *analysis.Pass, contextIn ast.Expr, contextOut *ast.Ident, funcName string) []byte {
	var contextInBuf strings.Builder
	printer.Fprint(&contextInBuf, pass.Fset, contextIn)

	// Wtf, where is this whitespace coming from???
	contextInSrc := strings.ReplaceAll(strings.ReplaceAll(contextInBuf.String(), "\t", ""), "\n", "")

	switch tracerStyle {
	case TracerStyleFunc:
		return []byte(fmt.Sprintf(`%s, span := %s().Start(%s, %q)
defer span.End()

`, contextOut.Name, tracerName, contextInSrc, funcName))
	case TracerStyleConst:
		return []byte(fmt.Sprintf(`%s, span := otel.Tracer(%s).Start(%s, %q)
defer span.End()

`, contextOut.Name, tracerName, contextInSrc, funcName))
	default:
		panic("unreachable")
	}
}

type astVisitorFunc func(ast.Node)

func (fn astVisitorFunc) Visit(node ast.Node) ast.Visitor {
	fn(node)
	return fn
}

func mustParseExpr(src string) ast.Expr {
	expr, err := parser.ParseExpr(src)
	if err != nil {
		panic(err)
	}
	return expr
}

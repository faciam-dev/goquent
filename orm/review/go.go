package review

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/faciam-dev/goquent/orm/query"
)

type chainCall struct {
	Method string
	Call   *ast.CallExpr
}

func reviewGoFile(path string) ([]Finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var findings []Finding
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		loc := sourceLocation(fset, path, sel.Sel.Pos())
		if isRawSQLMethod(sel.Sel.Name) {
			findings = append(findings, reviewRawSQLCall(sel, call, loc)...)
		}
		if isGoquentTerminal(sel.Sel.Name) {
			findings = append(findings, reviewGoquentChain(sel, call, loc)...)
		}
		if isRawBuilderMethod(sel.Sel.Name) {
			findings = append(findings, reviewRawBuilderCall(sel, call, loc)...)
		}
		return true
	})

	return applyFileSuppressions(path, findings)
}

func reviewRawSQLCall(sel *ast.SelectorExpr, call *ast.CallExpr, loc *query.SourceLocation) []Finding {
	argIndex := rawSQLArgIndex(sel.Sel.Name)
	if argIndex < 0 || argIndex >= len(call.Args) {
		return nil
	}
	sqlText, ok := stringLiteralValue(call.Args[argIndex])
	if !ok {
		if sel.Sel.Name == "RawPlan" || receiverLooksDatabase(sel.X) {
			return []Finding{staticFinding(
				query.WarningStaticReviewUnsupported,
				query.RiskMedium,
				"dynamic raw SQL could not be inspected",
				"prefer Goquent query builders or emit QueryPlan JSON from tests",
				loc,
				query.AnalysisUnsupported,
			)}
		}
		return nil
	}
	if !looksLikeSQL(sqlText) {
		return nil
	}
	return findingsFromPlan(query.NewRawPlan(sqlText), query.AnalysisPrecise, loc)
}

func reviewGoquentChain(sel *ast.SelectorExpr, call *ast.CallExpr, loc *query.SourceLocation) []Finding {
	calls, root, _ := collectChainCalls(call)
	if !chainHasRootBuilder(calls) {
		if receiverLooksQuery(sel.X) || receiverLooksQuery(root) {
			return []Finding{staticReviewPartial(loc)}
		}
		return nil
	}

	method := sel.Sel.Name
	var findings []Finding
	switch method {
	case "Update", "PlanUpdate":
		if !chainHasPredicate(calls) {
			findings = append(findings, staticFinding(
				query.WarningUpdateWithoutWhere,
				query.RiskBlocked,
				"UPDATE query has no WHERE predicate",
				"add a specific predicate before executing the update",
				loc,
				query.AnalysisPrecise,
			))
		} else if !chainHasPrimaryKeyLikePredicate(calls) {
			findings = append(findings, staticFinding(
				query.WarningBulkUpdateDetected,
				query.RiskMedium,
				"UPDATE predicate is not primary-key-like and may affect multiple rows",
				"confirm the intended row set or add a narrower predicate",
				loc,
				query.AnalysisPrecise,
			))
		}
	case "Delete", "PlanDelete":
		if !chainHasPredicate(calls) {
			findings = append(findings, staticFinding(
				query.WarningDeleteWithoutWhere,
				query.RiskBlocked,
				"DELETE query has no WHERE predicate",
				"add a specific predicate before executing the delete",
				loc,
				query.AnalysisPrecise,
			))
		} else if !chainHasPrimaryKeyLikePredicate(calls) {
			findings = append(findings, staticFinding(
				query.WarningBulkDeleteDetected,
				query.RiskMedium,
				"DELETE predicate is not primary-key-like and may affect multiple rows",
				"confirm the intended row set or add a narrower predicate",
				loc,
				query.AnalysisPrecise,
			))
		}
	case "Get", "GetMaps", "First", "FirstMap", "Plan":
		if !chainHasSelect(calls) {
			findings = append(findings, staticFinding(
				query.WarningSelectStarUsed,
				query.RiskMedium,
				"SELECT * makes selected data harder to review",
				"select explicit columns",
				loc,
				query.AnalysisPrecise,
			))
		}
		if !chainHasLimit(calls) && !chainIsAggregate(calls) {
			findings = append(findings, staticFinding(
				query.WarningLimitMissing,
				query.RiskMedium,
				"SELECT query has no LIMIT",
				"add Limit(n) for list queries",
				loc,
				query.AnalysisPrecise,
			))
		}
	}
	return findings
}

func reviewRawBuilderCall(sel *ast.SelectorExpr, call *ast.CallExpr, loc *query.SourceLocation) []Finding {
	if len(call.Args) == 0 {
		return nil
	}
	sqlText, ok := stringLiteralValue(call.Args[0])
	if !ok {
		return []Finding{staticFinding(
			query.WarningStaticReviewUnsupported,
			query.RiskMedium,
			"dynamic raw SQL fragment could not be inspected",
			"prefer structured predicates or keep raw SQL fragments literal and reviewed",
			loc,
			query.AnalysisUnsupported,
		)}
	}
	if normalizedContainsWeakPredicate(sqlText) {
		return []Finding{staticFinding(
			query.WarningWeakPredicate,
			query.RiskHigh,
			"query contains a weak predicate such as 1=1",
			"replace weak predicates with a meaningful filter",
			loc,
			query.AnalysisPrecise,
		)}
	}
	return nil
}

func staticReviewPartial(loc *query.SourceLocation) Finding {
	return staticFinding(
		query.WarningStaticReviewPartial,
		query.RiskMedium,
		"could not fully reconstruct Goquent query chain",
		"run query.Plan(ctx) in tests or keep the query chain inline for review",
		loc,
		query.AnalysisPartial,
	)
}

func staticFinding(code string, level query.RiskLevel, message, hint string, loc *query.SourceLocation, precision query.AnalysisPrecision) Finding {
	return Finding{
		Code:              code,
		Level:             level,
		Message:           message,
		Location:          cloneLocation(loc),
		Hint:              hint,
		AnalysisPrecision: precision,
	}
}

func sourceLocation(fset *token.FileSet, path string, pos token.Pos) *query.SourceLocation {
	p := fset.Position(pos)
	return &query.SourceLocation{File: path, Line: p.Line, Column: p.Column}
}

func collectChainCalls(expr ast.Expr) ([]chainCall, ast.Expr, bool) {
	var calls []chainCall
	for expr != nil {
		switch e := expr.(type) {
		case *ast.CallExpr:
			sel, ok := e.Fun.(*ast.SelectorExpr)
			if !ok {
				return calls, expr, false
			}
			calls = append(calls, chainCall{Method: sel.Sel.Name, Call: e})
			expr = sel.X
		case *ast.SelectorExpr:
			calls = append(calls, chainCall{Method: e.Sel.Name})
			expr = e.X
		case *ast.ParenExpr:
			expr = e.X
		case *ast.Ident:
			return calls, e, true
		default:
			return calls, expr, false
		}
	}
	return calls, nil, false
}

func chainHasRootBuilder(calls []chainCall) bool {
	for _, call := range calls {
		switch call.Method {
		case "Table", "Model":
			return true
		}
	}
	return false
}

func chainHasPredicate(calls []chainCall) bool {
	for _, call := range calls {
		if isPredicateMethod(call.Method) {
			return true
		}
	}
	return false
}

func chainHasPrimaryKeyLikePredicate(calls []chainCall) bool {
	for _, call := range calls {
		if !isPredicateMethod(call.Method) || call.Call == nil || len(call.Call.Args) == 0 {
			continue
		}
		col, ok := stringLiteralValue(call.Call.Args[0])
		if !ok {
			continue
		}
		if isPrimaryKeyLikeColumn(col) {
			return true
		}
	}
	return false
}

func chainHasSelect(calls []chainCall) bool {
	for _, call := range calls {
		switch call.Method {
		case "Select", "SelectRaw", "Distinct", "Count", "Max", "Min", "Sum", "Avg":
			return true
		}
	}
	return false
}

func chainHasLimit(calls []chainCall) bool {
	for _, call := range calls {
		switch call.Method {
		case "Limit", "Take":
			return true
		}
	}
	return false
}

func chainIsAggregate(calls []chainCall) bool {
	for _, call := range calls {
		switch call.Method {
		case "Count", "Max", "Min", "Sum", "Avg":
			return true
		}
	}
	return false
}

func isPredicateMethod(method string) bool {
	if strings.Contains(method, "Where") {
		return true
	}
	switch method {
	case "Having", "HavingRaw", "OrHaving", "OrHavingRaw":
		return true
	default:
		return false
	}
}

func isRawSQLMethod(method string) bool {
	switch method {
	case "Exec", "ExecContext", "Query", "QueryContext", "QueryRow", "QueryRowContext", "RawPlan":
		return true
	default:
		return false
	}
}

func rawSQLArgIndex(method string) int {
	switch method {
	case "Exec", "Query", "QueryRow":
		return 0
	case "ExecContext", "QueryContext", "QueryRowContext", "RawPlan":
		return 1
	default:
		return -1
	}
}

func isGoquentTerminal(method string) bool {
	switch method {
	case "Get", "GetMaps", "First", "FirstMap", "Plan", "Update", "PlanUpdate", "Delete", "PlanDelete":
		return true
	default:
		return false
	}
}

func isRawBuilderMethod(method string) bool {
	switch method {
	case "SelectRaw", "WhereRaw", "OrWhereRaw", "SafeWhereRaw", "SafeOrWhereRaw", "HavingRaw", "OrHavingRaw", "OrderByRaw":
		return true
	default:
		return false
	}
}

func stringLiteralValue(expr ast.Expr) (string, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", false
		}
		value, err := strconv.Unquote(e.Value)
		if err != nil {
			return "", false
		}
		return value, true
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", false
		}
		left, ok := stringLiteralValue(e.X)
		if !ok {
			return "", false
		}
		right, ok := stringLiteralValue(e.Y)
		if !ok {
			return "", false
		}
		return left + right, true
	case *ast.ParenExpr:
		return stringLiteralValue(e.X)
	default:
		return "", false
	}
}

func looksLikeSQL(s string) bool {
	upper := strings.ToUpper(s)
	for _, token := range []string{"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "TRUNCATE", "ALTER", "WITH"} {
		if containsSQLWord(upper, token) {
			return true
		}
	}
	return false
}

func isPrimaryKeyLikeColumn(column string) bool {
	col := strings.ToLower(strings.TrimSpace(column))
	col = strings.Trim(col, "`\"")
	return col == "id" || strings.HasSuffix(col, ".id") || strings.HasSuffix(col, "_id")
}

func receiverLooksDatabase(expr ast.Expr) bool {
	name := receiverName(expr)
	switch name {
	case "db", "sqlDB", "database", "tx", "conn", "executor":
		return true
	default:
		return strings.HasSuffix(strings.ToLower(name), "db")
	}
}

func receiverLooksQuery(expr ast.Expr) bool {
	name := strings.ToLower(receiverName(expr))
	if name == "q" || name == "qb" {
		return true
	}
	return strings.Contains(name, "query") || strings.Contains(name, "builder")
}

func receiverName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.CallExpr:
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			return sel.Sel.Name
		}
	}
	return ""
}

func normalizedContainsWeakPredicate(s string) bool {
	normalized := strings.ToLower(s)
	normalized = strings.ReplaceAll(normalized, "`", "")
	normalized = strings.ReplaceAll(normalized, `"`, "")
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "\t", "")
	normalized = strings.ReplaceAll(normalized, "\n", "")
	return strings.Contains(normalized, "where1=1") || normalized == "1=1" || strings.Contains(normalized, "(1=1)")
}

func containsSQLWord(upperSQL, token string) bool {
	for i := 0; i+len(token) <= len(upperSQL); i++ {
		if upperSQL[i:i+len(token)] != token {
			continue
		}
		beforeOK := i == 0 || !isSQLWordByte(upperSQL[i-1])
		after := i + len(token)
		afterOK := after >= len(upperSQL) || !isSQLWordByte(upperSQL[after])
		if beforeOK && afterOK {
			return true
		}
	}
	return false
}

func isSQLWordByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

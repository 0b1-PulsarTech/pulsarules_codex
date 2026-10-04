package packagevar

import "go/ast"

func (scan enumScan) exemptHandles(exempt map[string]bool) {
	for name, vs := range scan.vars {
		call, ok := vs.Values[0].(*ast.CallExpr)
		if name == "_" || !ok || !scan.constArgs(call) {
			continue
		}
		recv := scan.calleeRecv(call)
		if scan.isDefinition(scan.vars[recv]) {
			exempt[name] = true
			exempt[recv] = true
		}
	}
}

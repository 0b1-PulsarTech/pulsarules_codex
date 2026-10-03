package singleletter

type kind string

const (
	kindFunction      kind = "function"
	kindMethod        kind = "method"
	kindType          kind = "type"
	kindConstant      kind = "constant"
	kindVariable      kind = "variable"
	kindField         kind = "field"
	kindTypeParam     kind = "type parameter"
	kindPackage       kind = "package"
	kindLabel         kind = "label"
	kindReceiver      kind = "receiver"
	kindLoopVariable  kind = "loop variable"
	kindParameter     kind = "parameter"
	kindResult        kind = "result"
	kindSwitchBinding kind = "type switch binding"
	kindImportAlias   kind = "import alias"
)

func (siteKind kind) canBeExported() bool {
	switch siteKind {
	case kindFunction, kindMethod, kindType, kindConstant, kindVariable, kindField:
		return true
	case kindTypeParam, kindPackage, kindLabel, kindReceiver, kindLoopVariable,
		kindParameter, kindResult, kindSwitchBinding, kindImportAlias:
		return false
	}
	return false
}

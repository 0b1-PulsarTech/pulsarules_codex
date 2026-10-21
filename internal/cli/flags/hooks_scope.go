package flags

import "fmt"

type HooksScope string

func ParseHooksScope(raw string) (HooksScope, error) {
	scope := HooksScope(raw)
	if scope.Files() == nil {
		return "", fmt.Errorf("invalid --hooks-scope %q (want project|local)", raw)
	}
	return scope, nil
}

func (scope HooksScope) Files() []string {
	const project, local = "settings.json", "settings.local.json"
	switch scope {
	// Uninstall has no record of which scope install used, so empty lists both.
	case "":
		return []string{project, local}
	case "project":
		return []string{project}
	case "local":
		return []string{local}
	default:
		return nil
	}
}

func (scope HooksScope) File() string {
	// An empty scope lists the project file first, which is what install writes.
	return scope.Files()[0]
}

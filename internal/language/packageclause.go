package language

import "strings"

func isGoPackage(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "package ")
}

func isSQLSchema(line string) bool {
	// CREATE SCHEMA is SQL's closest analog to Go's package clause.
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(line)), "CREATE SCHEMA")
}

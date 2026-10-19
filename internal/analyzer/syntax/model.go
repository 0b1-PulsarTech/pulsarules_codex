package syntax

import (
	"context"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

type FunctionKind string

const (
	FuncFunction    FunctionKind = "function"
	FuncMethod      FunctionKind = "method"
	FuncConstructor FunctionKind = "constructor"
	FuncAccessor    FunctionKind = "accessor"
	FuncClosure     FunctionKind = "closure"
)

type NameKind string

const (
	NameFunction       NameKind = "function"
	NameMethod         NameKind = "method"
	NameType           NameKind = "type"
	NameConstant       NameKind = "constant"
	NameVariable       NameKind = "variable"
	NameField          NameKind = "field"
	NameTypeParam      NameKind = "type parameter"
	NamePackage        NameKind = "package"
	NameLabel          NameKind = "label"
	NameReceiver       NameKind = "receiver"
	NameLoopVariable   NameKind = "loop variable"
	NameParameter      NameKind = "parameter"
	NameResult         NameKind = "result"
	NameSwitchBinding  NameKind = "type switch binding"
	NameImportAlias    NameKind = "import alias"
	NameCatchParameter NameKind = "catch parameter"
	NameEnumValue      NameKind = "enum value"
)

type DocKind string

const (
	DocFunction    DocKind = "function"
	DocType        DocKind = "type"
	DocMethod      DocKind = "method"
	DocConstructor DocKind = "constructor"
	DocField       DocKind = "field"
	DocVariable    DocKind = "variable"
)

type File struct {
	Path, ParseError string
	Language         language.ID
	Functions        []Function
	Blocks           []Block
	Imports          []Import
	ImportGroups     []string
	Names            []Name
	DocSites         []DocSite
}

type Function struct {
	Name                     string
	Kind                     FunctionKind
	Line, BodyStart, BodyEnd int
	Params, NamedParams      int
	Complexity               int
	TestContainer, Nested    bool
}

type Block struct{ Line, Depth int }

type Import struct {
	Literal, Prefer string
	Line, Group     int
	Upward, Foreign bool
}

type Name struct {
	Name             string
	Kind             NameKind
	Line             int
	Exported, Exempt bool
}

type DocSite struct {
	Symbol                 string
	Kind                   DocKind
	Line                   int
	Exempt, MixedDirective bool
}

type Source struct {
	Path   string
	IsTest bool
}

type Provider interface {
	Language() language.ID
	Handles(ext string) bool
	Summarize(ctx context.Context, corpus Corpus) (map[string]*File, error)
}

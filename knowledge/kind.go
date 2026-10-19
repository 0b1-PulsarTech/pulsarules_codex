package knowledge

type Kind string

const (
	KindRule     Kind = "rules"
	KindPattern  Kind = "patterns"
	KindWorkflow Kind = "workflows"
	KindSkill    Kind = "skills"
)

const RouterSkillID = "project-router"

func (kind Kind) key(id string) string { return string(kind) + ":" + id }

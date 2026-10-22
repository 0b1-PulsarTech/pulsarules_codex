package command

import "flag"

type Project struct {
	dir string
}

func (project *Project) BindProject(fs *flag.FlagSet, usage string) {
	fs.StringVar(&project.dir, "project", "", usage)
}

func (project *Project) ProjectDir() string {
	if project.dir == "" {
		return "."
	}
	return project.dir
}

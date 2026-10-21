package install

import "io/fs"

type Context struct {
	Dir                 string
	Templates           fs.FS
	SettingsFile        string
	GitHooks            []string
	TypographicSeverity string
	BranchExtraTypes    string
}

type UninstallContext struct {
	Dir          string
	SettingsFile string
}

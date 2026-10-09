package command

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

const (
	dartPubspec    = "pubspec.yaml"
	dartPackageCfg = ".dart_tool/package_config.json"
)

func dartPrecheck(root, dir string, listed listing) string {
	for _, pubspecDir := range pubspecDirs(root, dir, listed) {
		if hasPackageConfig(root, pubspecDir) {
			continue
		}
		shown := "."
		if rel, err := filepath.Rel(root, pubspecDir); err == nil {
			shown = filepath.ToSlash(rel)
		}
		return fmt.Sprintf(
			"%s in %s has no %s; run `dart pub get` (then `dart run build_runner build`)",
			dartPubspec, shown, dartPackageCfg,
		)
	}
	return ""
}

func pubspecDirs(root, dir string, listed listing) []string {
	if listed == nil {
		if _, err := os.Stat(filepath.Join(dir, dartPubspec)); err != nil {
			return nil
		}
		return []string{dir}
	}
	prefix := ""
	if rel, err := filepath.Rel(root, dir); err == nil && rel != "." {
		prefix = filepath.ToSlash(rel) + "/"
	}
	var dirs []string
	for file := range listed {
		if path.Base(file) == dartPubspec && strings.HasPrefix(file, prefix) {
			dirs = append(dirs, filepath.Join(root, filepath.FromSlash(path.Dir(file))))
		}
	}
	slices.Sort(dirs)
	return dirs
}

func hasPackageConfig(root, dir string) bool {
	for {
		config := filepath.Join(dir, filepath.FromSlash(dartPackageCfg))
		if stat, err := os.Stat(config); err == nil && stat.Mode().IsRegular() {
			return true
		}
		if dir == root || dir == filepath.Dir(dir) {
			return false
		}
		dir = filepath.Dir(dir)
	}
}

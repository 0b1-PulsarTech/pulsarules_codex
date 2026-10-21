package mcpserver

import (
	"os/exec"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

type Server struct {
	Name    string
	Command string
	Args    []string
}

func managedServersByLanguage() map[language.ID][]Server {
	return map[language.ID][]Server{
		language.Go:   {{Name: "gopls", Command: "gopls", Args: []string{"mcp"}}},
		language.Dart: {{Name: "dart", Command: "dart", Args: []string{"mcp-server"}}},
	}
}

func ManagedNames() []string {
	var names []string
	for _, servers := range managedServersByLanguage() {
		for _, srv := range servers {
			names = append(names, srv.Name)
		}
	}
	slices.Sort(names)
	return names
}

func IsManaged(name, command string) bool {
	for _, servers := range managedServersByLanguage() {
		for _, srv := range servers {
			if srv.Name == name && srv.Command == command {
				return true
			}
		}
	}
	return false
}

func serversFor(languages []language.ID) []Server {
	var servers []Server
	managed := managedServersByLanguage()
	for _, lang := range languages {
		servers = append(servers, managed[lang]...)
	}
	return servers
}

func Available(languages []language.ID) (available []Server, skippedNames []string) {
	for _, srv := range serversFor(languages) {
		if _, lookErr := exec.LookPath(srv.Command); lookErr == nil {
			available = append(available, srv)
		} else {
			skippedNames = append(skippedNames, srv.Name)
		}
	}
	return available, skippedNames
}

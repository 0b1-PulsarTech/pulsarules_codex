package codex

import "testing"

func TestFindClash(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		want string
	}{
		{"absent", "[mcp_servers.goland]\ncommand = \"x\"\n", ""},
		{"other prefix only", "[mcp_servers.gopls2]\n", ""},
		{"table", "[mcp_servers.gopls]\n", "[mcp_servers.gopls]"},
		{"quoted table", "[mcp_servers.\"gopls\"]\n", "[mcp_servers.gopls]"},
		{"spaced table", "  [ mcp_servers . gopls ]  # mine\n", "[mcp_servers.gopls]"},
		{"sub table", "[mcp_servers.gopls.env]\n", "[mcp_servers.gopls]"},
		{"dotted key", "mcp_servers.gopls.command = \"x\"\n", "mcp_servers.gopls as a dotted key"},
		{
			"dotted inline",
			"mcp_servers.gopls = { command = \"x\" }\n",
			"mcp_servers.gopls as a dotted key",
		},
		{
			"inline parent",
			"mcp_servers = { dart = { command = \"x\" } }\n",
			"an inline mcp_servers table, which cannot be extended",
		},
		{
			"bare parent with gopls",
			"[mcp_servers]\ngopls = { command = \"x\" }\n",
			"gopls under [mcp_servers]",
		},
		{
			"bare parent with gopls table",
			"[mcp_servers]\ngopls.command = \"x\"\n",
			"gopls under [mcp_servers]",
		},
		{"bare parent without gopls", "[mcp_servers]\ndart = { command = \"x\" }\n", ""},
		{"bare parent alone", "[mcp_servers] # mine\n", ""},
		{"gopls key under another table", "[mcp_servers]\n[other]\ngopls = 1\n", ""},
		{"commented out", "# [mcp_servers.gopls]\n", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := findClash(testCase.text, "gopls"); got != testCase.want {
				t.Errorf("findClash = %q, want %q", got, testCase.want)
			}
		})
	}
}

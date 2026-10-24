package codex

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/mcpserver"
)

func goplsServer() mcpserver.Server {
	return mcpserver.Server{Name: "gopls", Command: "gopls", Args: []string{"mcp"}}
}

const goplsBlock = blockBegin + "\n[mcp_servers.gopls]\ncommand = \"gopls\"\n" +
	"args = [\"mcp\"]\n" + blockEnd + "\n"

type applyCase struct {
	name        string
	text        string
	servers     []mcpserver.Server
	want        string
	wantWired   []string
	wantClashed []string
}

func runApplyCases(t *testing.T, testCases []applyCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := applyBlock(testCase.text, testCase.servers)
			if err != nil {
				t.Fatalf("applyBlock: %v", err)
			}
			if got.text != testCase.want {
				t.Errorf("text = %q, want %q", got.text, testCase.want)
			}
			if !slices.Equal(got.wired, testCase.wantWired) {
				t.Errorf("wired = %v, want %v", got.wired, testCase.wantWired)
			}
			if !slices.Equal(got.clashed, testCase.wantClashed) {
				t.Errorf("clashed = %v, want %v", got.clashed, testCase.wantClashed)
			}
		})
	}
}

func TestApplyBlock_Writes(t *testing.T) {
	t.Parallel()

	oldBlock := strings.ReplaceAll(goplsBlock, "gopls\"", "old\"")
	runApplyCases(t, []applyCase{
		{
			name: "new file", text: "", servers: []mcpserver.Server{goplsServer()},
			want: goplsBlock, wantWired: []string{"gopls"},
		},
		{
			name: "append after a newline", text: "model = \"x\"\n",
			servers: []mcpserver.Server{goplsServer()},
			want:    "model = \"x\"\n\n" + goplsBlock, wantWired: []string{"gopls"},
		},
		{
			name: "append without a trailing newline", text: "model = \"x\"",
			servers: []mcpserver.Server{goplsServer()},
			want:    "model = \"x\"\n\n" + goplsBlock, wantWired: []string{"gopls"},
		},
		{
			name: "replace in place", text: "a = 1\n\n" + oldBlock + "b = 2\n",
			servers: []mcpserver.Server{goplsServer()},
			want:    "a = 1\n\n" + goplsBlock + "b = 2\n", wantWired: []string{"gopls"},
		},
		{
			name: "remove the block", text: "a = 1\n\n" + goplsBlock, servers: nil,
			want: "a = 1\n",
		},
	})
}

func TestApplyBlock_Clashes(t *testing.T) {
	t.Parallel()

	dart := mcpserver.Server{Name: "dart", Command: "dart", Args: []string{"mcp-server"}}
	runApplyCases(t, []applyCase{
		{
			name: "own table clashes", text: "[mcp_servers.gopls]\ncommand = \"mine\"\n",
			servers: []mcpserver.Server{goplsServer()},
			want:    "[mcp_servers.gopls]\ncommand = \"mine\"\n", wantClashed: []string{"gopls"},
		},
		{
			name: "clash drops our earlier block", text: goplsBlock + "[mcp_servers.gopls]\n",
			servers: []mcpserver.Server{goplsServer()},
			want:    "[mcp_servers.gopls]\n", wantClashed: []string{"gopls"},
		},
		{
			name:      "a parent table without gopls is no clash",
			text:      "[mcp_servers]\ndart = { command = \"d\" }\n",
			servers:   []mcpserver.Server{goplsServer()},
			want:      "[mcp_servers]\ndart = { command = \"d\" }\n\n" + goplsBlock,
			wantWired: []string{"gopls"},
		},
		{
			name: "one clashes and one is wired", text: "[mcp_servers.dart]\n",
			servers:   []mcpserver.Server{goplsServer(), dart},
			want:      "[mcp_servers.dart]\n\n" + goplsBlock,
			wantWired: []string{"gopls"}, wantClashed: []string{"dart"},
		},
	})
}

func TestApplyBlock_UnterminatedBlock(t *testing.T) {
	t.Parallel()

	text := "a = 1\n" + blockBegin + "\n[mcp_servers.gopls]\n"
	got, err := applyBlock(text, []mcpserver.Server{goplsServer()})
	if !errors.Is(err, ErrUnterminatedBlock) {
		t.Fatalf("err = %v, want ErrUnterminatedBlock", err)
	}
	if got.text != text {
		t.Errorf("text = %q, want it untouched", got.text)
	}
}

func TestTomlString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value string
		want  string
	}{
		{"plain", "gopls", `"gopls"`},
		{"quote and backslash", `a"b\c`, `"a\"b\\c"`},
		{"whitespace escapes", "a\nb\tc\rd", `"a\nb\tc\rd"`},
		{"control character", "a\x01b\x7f", `"a\u0001b\u007F"`},
		{"unicode passes through", "caf\u00e9", "\"caf\u00e9\""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := tomlString(testCase.value); got != testCase.want {
				t.Errorf("tomlString(%q) = %s, want %s", testCase.value, got, testCase.want)
			}
		})
	}
}

func TestTomlKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		key  string
		want string
	}{
		{"bare", "gopls-2_x", "gopls-2_x"},
		{"dot is quoted", "a.b", `"a.b"`},
		{"space is quoted", "a b", `"a b"`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := tomlKey(testCase.key); got != testCase.want {
				t.Errorf("tomlKey(%q) = %s, want %s", testCase.key, got, testCase.want)
			}
		})
	}
}

func TestRenderBlock_EscapesEveryField(t *testing.T) {
	t.Parallel()

	got := renderBlock([]mcpserver.Server{
		{Name: "my server", Command: `C:\bin\x`, Args: []string{`a"b`, "c"}},
	})
	want := blockBegin + "\n[mcp_servers.\"my server\"]\ncommand = \"C:\\\\bin\\\\x\"\n" +
		"args = [\"a\\\"b\", \"c\"]\n" + blockEnd + "\n"
	if got != want {
		t.Errorf("block = %q, want %q", got, want)
	}
	if renderBlock(nil) != "" {
		t.Errorf("no servers must render nothing")
	}
}

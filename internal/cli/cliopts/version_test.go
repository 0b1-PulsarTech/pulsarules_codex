package cliopts

import (
	"runtime/debug"
	"testing"
)

type versionCase struct {
	name  string
	build *debug.BuildInfo
	want  string
}

func versionCases() []versionCase {
	return []versionCase{
		{name: "no build info falls back", want: fallbackVersion},
		{
			name:  "released module reports its own version",
			build: &debug.BuildInfo{Main: debug.Module{Version: "v0.3.1"}},
			want:  "v0.3.1",
		},
		{
			name: "released module appends the trimmed revision",
			build: &debug.BuildInfo{
				Main: debug.Module{Version: "v0.3.1"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1bc4721e61aebeef0123"},
				},
			},
			want: "v0.3.1+1bc4721e61ae",
		},
		{
			name: "local build names itself devel rather than a stale number",
			build: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1bc4721e61aebeef0123"},
				},
			},
			want: "devel+1bc4721e61ae",
		},
		{
			name:  "local build with no revision falls back",
			build: &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			want:  fallbackVersion,
		},
		{
			name: "a short revision is not trimmed",
			build: &debug.BuildInfo{
				Main:     debug.Module{Version: "v0.3.1"},
				Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "1bc4721"}},
			},
			want: "v0.3.1+1bc4721",
		},
		{
			name: "a pseudo-version already carrying the revision is not doubled",
			build: &debug.BuildInfo{
				Main: debug.Module{Version: "v0.3.1-0.20260818221113-1bc4721e61ae"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1bc4721e61aebeef0123"},
				},
			},
			want: "v0.3.1-0.20260818221113-1bc4721e61ae",
		},
		{
			name: "an unrelated setting is ignored",
			build: &debug.BuildInfo{
				Main:     debug.Module{Version: "v0.3.1"},
				Settings: []debug.BuildSetting{{Key: "vcs.modified", Value: "true"}},
			},
			want: "v0.3.1",
		},
	}
}

func TestFormatVersion(t *testing.T) {
	t.Parallel()

	testCases := versionCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := formatVersion(testCase.build); got != testCase.want {
				t.Errorf("formatVersion() = %q, want %q", got, testCase.want)
			}
		})
	}
}

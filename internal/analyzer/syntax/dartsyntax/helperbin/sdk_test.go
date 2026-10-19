package helperbin

import (
	"strings"
	"testing"
)

func TestParseSDKVersion(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{name: "stable", output: "Dart SDK version: 3.13.2 (stable)", want: "3.13.2"},
		{name: "prerelease", output: "Dart SDK version: 3.12.0-dev.1 (dev)", want: "3.12.0-dev.1"},
		{name: "vm label", output: "Dart VM version: 3.12.0 (stable)", want: "3.12.0"},
		{name: "garbage", output: "nothing here", wantErr: true},
		{name: "analytics decoy alone", output: "Analytics version: 2.0", wantErr: true},
		{
			name:   "analytics decoy first",
			output: "Analytics version: 2.0\nDart SDK version: 3.13.2 (stable)", want: "3.13.2",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseSDKVersion(testCase.output)
			if (err != nil) != testCase.wantErr || got != testCase.want {
				t.Errorf("parseSDKVersion = %q, %v; want %q, wantErr %v",
					got, err, testCase.want, testCase.wantErr)
			}
		})
	}
}

func TestParseSDKVersionMessage(t *testing.T) {
	t.Parallel()
	_, err := parseSDKVersion("garbage 47\ngarbage 48")
	assertErrContains(t, err, `no version in "garbage 47\ngarbage 48"`)
}

func TestCheckSDK(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		version string
		wantErr string
	}{
		{name: "exact floor", version: "3.11.0"},
		{name: "newer minor", version: "3.13.2"},
		{name: "newer major", version: "4.0.0"},
		{name: "prerelease of a newer minor", version: "3.12.0-dev.1"},
		{
			name: "prerelease at floor", version: "3.11.0-123.dev",
			wantErr: "found 3.11.0-123.dev",
		},
		{name: "patch above floor", version: "3.11.1"},
		{name: "older minor", version: "3.10.9", wantErr: "found 3.10.9"},
		{name: "older major", version: "2.19.6", wantErr: "Dart SDK 3.11+ required, found 2.19.6"},
		{name: "unparsable", version: "weird", wantErr: "found weird"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := checkSDK(testCase.version)
			if testCase.wantErr == "" && err != nil {
				t.Fatalf("checkSDK = %v, want nil", err)
			}
			missing := err == nil || !strings.Contains(err.Error(), testCase.wantErr)
			if testCase.wantErr != "" && missing {
				t.Fatalf("checkSDK = %v, want containing %q", err, testCase.wantErr)
			}
		})
	}
}

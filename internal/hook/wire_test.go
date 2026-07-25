package hook

import "testing"

func TestDecodeHookPayload(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		payload      string
		wantSession  string
		wantFilePath string
	}{
		{
			name:         "well formed payload",
			payload:      `{"session_id":"s1","tool_input":{"file_path":"main.go"}}`,
			wantSession:  "s1",
			wantFilePath: "main.go",
		},
		{
			name:        "session without tool input",
			payload:     `{"session_id":"s2"}`,
			wantSession: "s2",
		},
		{
			name:    "malformed payload degrades to the zero value",
			payload: `{not json`,
		},
		{
			name:    "empty payload degrades to the zero value",
			payload: "",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			decoded := decodeHookPayload([]byte(testCase.payload))
			if decoded.SessionID != testCase.wantSession {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, testCase.wantSession)
			}
			if decoded.ToolInput.FilePath != testCase.wantFilePath {
				t.Errorf(
					"FilePath = %q, want %q",
					decoded.ToolInput.FilePath,
					testCase.wantFilePath,
				)
			}
		})
	}
}

package tsfacts

import "testing"

type callbackCase struct {
	name   string
	text   string
	isTest bool
	want   bool
}

func callbackCases() []callbackCase {
	return []callbackCase{
		{name: "describe", text: "describe('s', () => {});", isTest: true, want: true},
		{name: "it", text: "it('s', () => {});", isTest: true, want: true},
		{
			name:   "test function expression",
			text:   "test('s', function () {});",
			isTest: true,
			want:   true,
		},
		{name: "only variant", text: "describe.only('s', () => {});", isTest: true, want: true},
		{name: "skip variant", text: "it.skip('s', () => {});", isTest: true, want: true},
		{name: "hook", text: "beforeEach(() => {});", isTest: true, want: true},
		{name: "async callback", text: "it('s', async () => {});", isTest: true, want: true},
		{name: "not a test file", text: "describe('s', () => {});", isTest: false, want: false},
		{name: "other callee", text: "run('s', () => {});", isTest: true, want: false},
		{
			name:   "member of other object",
			text:   "a.b.describe(() => {});",
			isTest: true,
			want:   false,
		},
		{name: "assigned not passed", text: "const f = () => {};", isTest: true, want: false},
		{
			name:   "wrapped in a call argument list",
			text:   "it('s', wrap(() => {}));",
			isTest: true,
			want:   false,
		},
	}
}

func TestIsTestCallback(t *testing.T) {
	t.Parallel()

	testCases := callbackCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := false
			options := Options{IsTest: testCase.isTest}
			for _, fn := range collectOf(t, testCase.text, options).Functions {
				got = got || fn.TestContainer
			}
			if got != testCase.want {
				t.Fatalf("test container %v, want %v", got, testCase.want)
			}
		})
	}
}

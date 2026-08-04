package naming

import "testing"

// TestNumberedStem pins the split alone. Whether a numbered name is actually a
// sequence needs the file's other names, which fileChecker.isSequential adds.
func TestNumberedStem(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		ident      string
		wantStem   string
		wantDigits int
		wantOK     bool
	}{
		{"counter suffix", "foo1", "foo", 1, true},
		{"two-digit counter", "bar22", "bar", 22, true},
		{"no digit", "fooBar", "", 0, false},
		{"markdown heading level", "H1", "", 0, false},
		{"acronym with trailing digit", "UTF8", "", 0, false},
		{"version marker", "V2", "", 0, false},
		{"digit after uppercase mid-word", "bodyWithoutH1", "", 0, false},
		{"empty", "", "", 0, false},
		{"all digits", "123", "", 0, false},
		{"hash size", "sha256", "sha", 256, true},
		{"bit width", "asUint64", "asUint", 64, true},
		{"radix", "pow10", "pow", 10, true},
		{"domain magnitude", "PhoneTier250", "PhoneTier", 250, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			stem, digits, ok := numberedStem(testCase.ident)
			if ok != testCase.wantOK {
				t.Fatalf("numberedStem(%q) ok = %v, want %v", testCase.ident, ok, testCase.wantOK)
			}
			if stem != testCase.wantStem || digits != testCase.wantDigits {
				t.Fatalf(
					"numberedStem(%q) = (%q, %d), want (%q, %d)",
					testCase.ident, stem, digits, testCase.wantStem, testCase.wantDigits,
				)
			}
		})
	}
}

// TestCheckHungarian pins the distinction that made the rule useless: a type
// tag is LOWER case and glued to a capitalised noun, so an identifier starting
// upper case can never be Hungarian, however its acronym happens to begin.
func TestCheckHungarian(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		ident string
		want  bool
	}{
		{"string prefix", "strName", true},
		{"sz prefix", "szBuffer", true},
		{"count prefix", "nCount", true},
		{"bool prefix", "bFlag", true},
		{"handle prefix", "hWindow", true},
		{"unsigned prefix", "uCount", true},
		{"index prefix", "iIndex", true},
		{"acronym-initial domain noun", "IDSelector", false},
		{"acronym-initial interface", "HTTPDoer", false},
		{"acronym-initial header type", "HTTPHeader", false},
		{"bare acronym", "UUID", false},
		{"currency acronym", "BRL", false},
		{"vendor acronym", "WABAEntry", false},
		{"url acronym", "URLParser", false},
		{"acronym with trailing digit", "UTF8", false},
		{"plain camel case", "userName", false},
		{"prefix without a capital after it", "strings", false},
		{"lower-case acronym-ish word", "idSelector", false},
		{"single letter", "i", false},
		{"empty", "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := checkHungarian(testCase.ident); got != testCase.want {
				t.Fatalf("checkHungarian(%q) = %v, want %v", testCase.ident, got, testCase.want)
			}
		})
	}
}

func TestCheckNoiseWord(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		ident string
		want  bool
	}{
		{"exact noise word", "data", true},
		{"exact noise word capitalised", "Data", true},
		{"helper", "helper", true},
		{"manager", "manager", true},
		{"noise word as a prefix is specific enough", "dataValue", false},
		{"noise word as a prefix capitalised", "DataFile", false},
		{"temp prefix", "tempFile", false},
		{"tmp prefix", "tmpDir", false},
		{"base is a domain noun, not noise", "base", false},
		{"Base is a domain noun, not noise", "Base", false},
		{"ordinary name", "userName", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := checkNoiseWord(testCase.ident); got != testCase.want {
				t.Fatalf("checkNoiseWord(%q) = %v, want %v", testCase.ident, got, testCase.want)
			}
		})
	}
}

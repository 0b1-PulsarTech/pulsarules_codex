package loopvari

import "testing"

func TestClassify(t *testing.T) {
	t.Parallel()

	testCases := literalCases()
	runLoopCases(t, testCases)
}

func literalCases() []loopCase {
	return []loopCase{
		{
			name: "composite literal key is not a use",
			body: "type T struct{ i int }\nfor i := range s {\n _ = T{i: s[i]}\n}",
		},
		{
			name:   "composite literal value is a use",
			body:   "type T struct{ i int }\nfor i := range s {\n _ = T{i: i}\n}",
			expect: 1,
		},
		{name: "conversion inside an index", body: "for i := range s {\n _ = s[int(i)]\n}"},
		{
			name:   "conversion passed to a call",
			body:   "for i := range s {\n _ = fmt.Sprint(int(i))\n}",
			expect: 1,
		},
		{
			name:   "conversion stored",
			body:   "for i := range s {\n x := int64(i)\n _ = x\n}",
			expect: 1,
		},
		{
			name: "label named i is not a use",
			body: "for i := range s {\n _ = s[i]\ni:\n for range s {\n  break i\n }\n}",
		},
		{
			name: "continue to a label named i",
			body: "for i := range s {\n _ = s[i]\ni:\n for range s {\n  continue i\n }\n}",
		},
	}
}

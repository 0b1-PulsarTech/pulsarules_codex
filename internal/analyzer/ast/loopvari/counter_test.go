package loopvari

import (
	"slices"
	"testing"
)

func TestCountsInner(t *testing.T) {
	t.Parallel()

	runLoopCases(t, slices.Concat(baseCases(), shadowCases(), definitionCases()))
}

func baseCases() []loopCase {
	return []loopCase{
		{
			name:   "inner init alone does not move the counter",
			body:   "for i := range s {\n for j := 0; j < i; {\n  _ = s[j]\n  break\n }\n}",
			expect: 1,
		},
		{
			name: "shadowing define then step in the body",
			body: "for i := range s {\n j := 0\n for j < i {\n  j := 5\n  j++\n  break\n }\n" +
				" _ = j\n}",
			expect: 1,
		},
		{
			name: "shadowing range variable stepped",
			body: "for i := range s {\n j := 0\n for j < i {\n  for j := range s {\n   j++\n" +
				"   _ = j\n  }\n  break\n }\n _ = j\n}",
			expect: 1,
		},
		{
			name: "shadowing if init stepped",
			body: "for i := range s {\n j := 0\n for j < i {\n  if j := 1; j > 0 {\n   j++\n" +
				"  }\n  break\n }\n _ = j\n}",
			expect: 1,
		},
		{
			name: "shadowing func parameter stepped",
			body: "for i := range s {\n j := 0\n for j < i {\n  func(j int) {\n   j++\n" +
				"  }(0)\n  break\n }\n _ = j\n}",
			expect: 1,
		},
		{
			name: "shadowing var declaration stepped",
			body: "for i := range s {\n j := 0\n for j < i {\n  var j int\n  j++\n  break\n" +
				" }\n _ = j\n}",
			expect: 1,
		},
		{
			name: "step before a later shadowing define",
			body: "for i := range s {\n j := 0\n for j < i {\n  j++\n  j := 5\n  _ = j\n }\n}",
		},
		{
			name: "step inside a case body",
			body: "for i := range s {\n j := 0\n for j < i {\n  switch {\n  case true:\n   j++\n" +
				"  }\n }\n}",
		},
		{
			name:  "select case receive shadows the counter",
			decls: "var ch chan int",
			body: "for i := range s {\n j := 0\n for j < i {\n  select {\n  case j := <-ch:\n" +
				"   j++\n  }\n  break\n }\n _ = j\n}",
			expect: 1,
		},
		{
			name:  "step in a select case body",
			decls: "var ch chan int",
			body: "for i := range s {\n j := 0\n for j < i {\n  select {\n  case <-ch:\n" +
				"   j++\n  }\n }\n}",
		},
		{
			name: "plain assignment moves the counter",
			body: "for i := range s {\n j := 0\n for j < i {\n  j = j + 1\n }\n}",
		},
	}
}

func innerStep(stmt string) string {
	return "for i := range s {\n j := 0\n for j < i {\n  " + stmt + "\n  break\n }\n _ = j\n}"
}

func shadowCases() []loopCase {
	return []loopCase{
		{
			name:   "case clause body defines and steps its own counter",
			body:   innerStep("switch {\n  case true:\n   j := 1\n   j++\n  }"),
			expect: 1,
		},
		{
			name:   "switch init defines and steps its own counter",
			body:   innerStep("switch j := 1; {\n  default:\n   j++\n  }"),
			expect: 1,
		},
		{
			name:   "type switch init defines and steps its own counter",
			decls:  "var v any",
			body:   innerStep("switch j := 1; v.(type) {\n  default:\n   j++\n  }"),
			expect: 1,
		},
		{
			name:   "type switch binding stepped",
			decls:  "var v any",
			body:   innerStep("switch j := v.(type) {\n  case int:\n   j++\n  }"),
			expect: 1,
		},
		{
			name: "range assigning the counter without define moves it",
			body: innerStep("for j = range s {\n   j++\n  }"),
		},
		{
			name:   "range value define stepped",
			body:   innerStep("for _, j := range s {\n   j++\n   _ = j\n  }"),
			expect: 1,
		},
	}
}

func definitionCases() []loopCase {
	return []loopCase{
		{
			name:  "select case assigns the counter",
			decls: "var ch chan int",
			body:  innerStep("select {\n  case j = <-ch:\n  }"),
		},
		{
			name:   "select default clause does not move the counter",
			decls:  "var ch chan int",
			body:   innerStep("select {\n  default:\n  }"),
			expect: 1,
		},
		{
			name:  "select case define reads the outer counter on its right",
			decls: "var ch chan int",
			body: innerStep(
				"select {\n  case j := <-func() chan int {\n   j++\n   return ch\n  }():\n" +
					"   _ = j\n  }",
			),
		},
		{
			name: "case expression steps the counter",
			body: innerStep("switch {\n  case func() bool {\n   j++\n   return true\n  }():\n  }"),
		},
		{
			name: "define right side steps the outer counter",
			body: innerStep("j := func() int {\n   j++\n   return j\n  }()\n  _ = j"),
		},
		{
			name: "var value steps the outer counter",
			body: innerStep("var j = func() int {\n   j++\n   return j\n  }()\n  _ = j"),
		},
		{
			name: "var spec before the shadow steps the outer counter",
			body: innerStep(
				"var (\n   k = func() int {\n    j++\n    return 1\n   }()\n   j = 2\n  )\n" +
					"  _, _ = j, k",
			),
		},
		{
			name:   "define right side is a literal whose parameter shadows",
			body:   innerStep("j := func(j int) {\n   j++\n  }\n  _ = j"),
			expect: 1,
		},
		{
			name:   "define right side without a step stays shadowed",
			body:   innerStep("j := 5\n  j++\n  _ = j"),
			expect: 1,
		},
	}
}

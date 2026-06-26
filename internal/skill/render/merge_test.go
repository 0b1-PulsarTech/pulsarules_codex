package render

import "testing"

func TestBodySections(t *testing.T) {
	t.Parallel()

	body := "preamble text\n" +
		"{{define \"must\"}}\nMUST BODY\n{{end}}\n" +
		"{{define \"forbidden\"}}\nFORBIDDEN BODY\n{{end}}"
	sections, err := bodySections(body)
	if err != nil {
		t.Fatalf("bodySections: %v", err)
	}
	if sections["must"] != "MUST BODY" {
		t.Errorf("must = %q, want MUST BODY", sections["must"])
	}
	if sections["forbidden"] != "FORBIDDEN BODY" {
		t.Errorf("forbidden = %q, want FORBIDDEN BODY", sections["forbidden"])
	}
	if _, ok := sections["recipe"]; ok {
		t.Error("recipe should be absent")
	}
}

// TestMergeSections asserts same-keyed sections from multiple sources group under
// one heading in source order, and headings appear in canonical order.
func TestMergeSections(t *testing.T) {
	t.Parallel()

	sources := []source{
		{name: "Effective Go", sections: map[string]string{"must": "a-must"}},
		{name: "Naming", sections: map[string]string{"must": "b-must", "forbidden": "b-forbidden"}},
	}
	merged := mergeSections(sources)
	if len(merged) != 2 {
		t.Fatalf("expected Must + Forbidden, got %d sections", len(merged))
	}
	if merged[0].Heading != "Must" || len(merged[0].Items) != 2 {
		t.Errorf("first section = %+v, want Must with 2 items", merged[0])
	}
	if merged[0].Items[0].Name != "Effective Go" || merged[0].Items[1].Name != "Naming" {
		t.Errorf("Must items out of source order: %+v", merged[0].Items)
	}
	if merged[1].Heading != "Forbidden" || len(merged[1].Items) != 1 {
		t.Errorf("second section = %+v, want Forbidden with 1 item", merged[1])
	}
}

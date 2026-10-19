package sfc

type Region struct {
	Tag        string
	Start, End int
}

func Regions(text string) []Region { return regionsOf(&markup{text: text}) }

func PlainRegions(text string) []Region {
	// Vue markup gives braces no meaning, so a stray one must not hide a later script.
	return regionsOf(&markup{text: text, plain: true})
}

func regionsOf(scan *markup) []Region {
	scan.lower = asciiLower(scan.text)
	// A script left unterminated still has its region, so the error is not the caller's.
	_ = scan.run()
	return scan.regions
}

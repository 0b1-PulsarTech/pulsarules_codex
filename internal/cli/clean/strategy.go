package clean

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible/cleaner"

type sweepStrategy struct {
	apply   func(scrub *cleaner.Cleaner, path string) (cleaner.Report, error)
	verb    string
	summary func(tally cleaner.Tally) string
}

func inspecting() sweepStrategy {
	return sweepStrategy{
		apply:   (*cleaner.Cleaner).Inspect,
		verb:    "found",
		summary: cleaner.Tally.ReportLine,
	}
}

func rewriting() sweepStrategy {
	return sweepStrategy{
		apply:   (*cleaner.Cleaner).CleanFile,
		verb:    "left",
		summary: cleaner.Tally.WriteLine,
	}
}

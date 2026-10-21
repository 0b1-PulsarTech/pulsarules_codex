package ownedfile

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/report"

type Ledger struct {
	BackedUp []string
	Restored []string
	Orphaned []string
}

func (ledger *Ledger) AddBackup(note string) {
	if note != "" {
		ledger.BackedUp = append(ledger.BackedUp, note)
	}
}

func (ledger *Ledger) addRestore(note string) {
	if note != "" {
		ledger.Restored = append(ledger.Restored, note)
	}
}

func (ledger Ledger) Report() report.Report {
	var rpt report.Report
	for _, msg := range ledger.BackedUp {
		rpt.Warn("%s", msg)
	}
	for _, msg := range ledger.Orphaned {
		rpt.Warn("%s", msg)
	}
	for _, msg := range ledger.Restored {
		rpt.Note("%s", msg)
	}
	return rpt
}

package domain

import (
	"errors"
	"testing"
)

func TestNormalizeExportSections_IncludesCustomSignals(t *testing.T) {
	all, err := NormalizeExportSections(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != len(AllExportSections) {
		t.Fatalf("default sections=%v", all)
	}
	found := false
	for _, s := range all {
		if s == ExportSectionCustomSignals {
			found = true
		}
	}
	if !found {
		t.Fatalf("default missing customSignals: %v", all)
	}
	got, err := NormalizeExportSections([]string{"customSignals", "CUSTOMSIGNALS"})
	if err != nil || len(got) != 1 || got[0] != ExportSectionCustomSignals {
		t.Fatalf("%v %v", got, err)
	}
	_, err = NormalizeExportSections([]string{"nope"})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("want invalid, got %v", err)
	}
}

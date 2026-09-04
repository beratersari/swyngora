package domain

import "testing"

func TestExplainForensics_JoinsTapeAndSimilar(t *testing.T) {
	got := ExplainForensics(ForensicsReport{
		Tape:    &AroundReport{Summary: "Price rose 2% during."},
		Similar: &AroundSimilarReport{Summary: "3 similar cases went up after."},
	})
	if got != "Price rose 2% during. 3 similar cases went up after." {
		t.Fatalf("%q", got)
	}
}

func TestExplainForensics_SimilarError(t *testing.T) {
	got := ExplainForensics(ForensicsReport{
		Tape:         &AroundReport{Summary: "Volume faded after."},
		SimilarError: "upstream timeout",
	})
	if got != "Volume faded after. Similar past moves could not be loaded." {
		t.Fatalf("%q", got)
	}
}

func TestExplainForensics_Empty(t *testing.T) {
	if got := ExplainForensics(ForensicsReport{}); got != "No forensics tape for this time." {
		t.Fatalf("%q", got)
	}
}

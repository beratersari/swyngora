package portfolio

import (
	"encoding/json"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestOrderPreviewPayload_IncludesReasonHistory(t *testing.T) {
	raw, err := json.Marshal(OrderPreviewPayload(&domain.OrderPreview{
		ReasonSet: true,
		Reason:    []string{domain.SituationClausePriceCVD},
		ReasonHistory: []domain.SituationLeaveRow{{
			Clause: domain.SituationClausePriceCVD,
			Kind:   domain.SituationCVDPriceUp,
			Still: domain.SituationLeaveBucket{
				Count: 1, Profitable: 1, RealizedPnL: 20,
				Closes: []domain.SituationRecordClose{{Symbol: "BTCUSDT", RealizedPnL: 20}},
			},
			Left: domain.SituationLeaveBucket{Closes: []domain.SituationRecordClose{}},
		}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, `"reasonHistory"`) || !strings.Contains(text, `"clause":"priceCvd"`) {
		t.Fatalf("%s", text)
	}
	if !strings.Contains(text, `"left":{"count":0,"profitable":0,"realizedPnL":0,"closes":[]}`) {
		t.Fatalf("empty left %s", text)
	}
}

func TestOrderPreviewPayload_OmitsReasonHistoryWhenQuiet(t *testing.T) {
	out := OrderPreviewPayload(&domain.OrderPreview{
		SituationStatus: domain.SituationPreviewReady,
		ReasonSet:       true,
		Reason:          []string{},
	})
	if _, ok := out["reasonHistory"]; ok {
		t.Fatalf("%+v", out["reasonHistory"])
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "reasonHistory") {
		t.Fatalf("%s", raw)
	}
}

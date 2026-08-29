package domain

import (
	"testing"
	"time"
)

func TestLatestDueRunAt_SkipsMissed(t *testing.T) {
	// Daily: next was Mon, now Thu → latest due is Thu (not Mon/Tue/Wed separately)
	mon := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday
	thu := time.Date(2024, 6, 6, 15, 0, 0, 0, time.UTC)
	got := LatestDueRunAt(mon, thu, RecurringDaily)
	want := time.Date(2024, 6, 6, 10, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	// Future next → unchanged
	fri := time.Date(2024, 6, 7, 10, 0, 0, 0, time.UTC)
	if g := LatestDueRunAt(fri, thu, RecurringDaily); !g.Equal(fri) {
		t.Fatalf("future: %v", g)
	}
}

func TestRecurringPeriodKey(t *testing.T) {
	at := time.Date(2024, 6, 5, 12, 0, 0, 0, time.UTC) // Wed week 23
	if k := RecurringPeriodKey(at, RecurringDaily); k != "2024-06-05" {
		t.Fatal(k)
	}
	if k := RecurringPeriodKey(at, RecurringMonthly); k != "2024-06" {
		t.Fatal(k)
	}
	if k := RecurringPeriodKey(at, RecurringWeekly); k != "2024-W23" {
		t.Fatal(k)
	}
}

func TestAdvanceRecurringRunAt(t *testing.T) {
	at := time.Date(2024, 1, 31, 12, 0, 0, 0, time.UTC)
	m := AdvanceRecurringRunAt(at, RecurringMonthly)
	if m.Month() != time.February || m.Day() != 29 && m.Year() == 2024 {
		// 2024 is leap: Jan 31 + 1 month = Feb 29 in Go AddDate
		if m.Month() != time.March && m.Month() != time.February {
			t.Fatalf("%v", m)
		}
	}
}

func TestRecurringIntervalAndSalaryDay(t *testing.T) {
	from := time.Date(2024, 6, 10, 9, 0, 0, 0, time.UTC)
	iv := RecurringBuyPlan{Frequency: RecurringInterval, IntervalHours: 12}
	next := AdvanceRecurringSchedule(from, iv)
	if !next.Equal(from.Add(12 * time.Hour)) {
		t.Fatalf("interval +12h: %v", next)
	}
	if k := RecurringPeriodKeyPlan(from, iv); k != "i12-1718002800" && k[:4] != "i12-" {
		t.Fatalf("interval key=%s", k)
	}
	// Salary day 31 from Jan 31 → Feb 29 2024
	jan := time.Date(2024, 1, 31, 8, 0, 0, 0, time.UTC)
	feb := AdvanceRecurringSchedule(jan, RecurringBuyPlan{Frequency: RecurringMonthly, DayOfMonth: 31})
	if feb.Month() != time.February || feb.Day() != 29 {
		t.Fatalf("clamp 31 → %v", feb)
	}
	// Next Monday on or after Wednesday
	wed := time.Date(2024, 6, 5, 10, 0, 0, 0, time.UTC) // Wednesday
	mon := AlignRecurringStart(wed, RecurringBuyPlan{Frequency: RecurringWeekly, Weekday: "monday"})
	if mon.Weekday() != time.Monday || mon.Day() != 10 {
		t.Fatalf("next monday=%v", mon)
	}
	now := time.Date(2024, 6, 26, 12, 0, 0, 0, time.UTC) // after the 15th
	first := FirstRecurringRunAt(now, nil, RecurringBuyPlan{Frequency: RecurringMonthly, DayOfMonth: 15})
	if first.Month() != time.July || first.Day() != 15 {
		t.Fatalf("salary next=%v", first)
	}
	name, err := NormalizeRecurringBuyName("Salary Day Buy", "BTCUSDT", RecurringMonthly)
	if err != nil || name != "Salary Day Buy" {
		t.Fatalf("%q %v", name, err)
	}
	def, err := NormalizeRecurringBuyName("", "ETHUSDT", RecurringInterval)
	if err != nil || def != "ETHUSDT interval" {
		t.Fatalf("default name %q %v", def, err)
	}
	if err := ValidateRecurringSchedule(RecurringInterval, "", 0, 12); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRecurringSchedule(RecurringWeekly, "monday", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRecurringSchedule(RecurringDaily, "monday", 0, 0); err == nil {
		t.Fatal("daily+weekday")
	}
}

func TestRecurringMondayNineIstanbul(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		t.Fatal(err)
	}
	// Wednesday afternoon UTC → next Monday 09:00 Istanbul
	wed := time.Date(2024, 6, 5, 15, 0, 0, 0, time.UTC)
	p := RecurringBuyPlan{
		Frequency: RecurringWeekly, Weekday: "monday",
		TimeZone: "Europe/Istanbul", HasLocalTime: true, Hour: 9, Minute: 0,
	}
	got := AlignRecurringStart(wed, p)
	want := time.Date(2024, 6, 10, 9, 0, 0, 0, loc).UTC()
	if !got.Equal(want) {
		t.Fatalf("align got %v want %v (local %v)", got, want, got.In(loc))
	}
	next := AdvanceRecurringSchedule(got, p)
	wantNext := time.Date(2024, 6, 17, 9, 0, 0, 0, loc).UTC()
	if !next.Equal(wantNext) {
		t.Fatalf("advance got %v want %v", next, wantNext)
	}
	if k := RecurringPeriodKeyPlan(got, p); k != "2024-W24" {
		t.Fatalf("period=%s", k)
	}
}

func TestRecurringMaxPriceBlocks(t *testing.T) {
	if got := RecurringMaxPriceBlocks(66000, 0.001, 0.001, 65000); got == "" {
		t.Fatal("last over max should block")
	}
	if got := RecurringMaxPriceBlocks(64000, 0.001, 0.001, 65000); got != "" {
		t.Fatalf("64k should buy: %s", got)
	}
	// last under max but slipped+fee crosses the cap
	if got := RecurringMaxPriceBlocks(64990, 0.001, 0.001, 65000); got == "" {
		t.Fatal("effective price over max should block")
	}
	if got := RecurringMaxPriceBlocks(64000, 0.001, 0.001, 0); got != "" {
		t.Fatalf("no cap: %s", got)
	}
	if _, err := NormalizeRecurringTimeZone("Europe/Istanbul"); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeRecurringTimeZone("Not/AZone"); err == nil {
		t.Fatal("expected invalid timezone")
	}
	if v, err := ResolveRecurringMaxPrice(65000); err != nil || v != 65000 {
		t.Fatalf("%v %v", v, err)
	}
	if _, err := ResolveRecurringMaxPrice(-1); err == nil {
		t.Fatal("expected invalid maxPrice")
	}
}

func TestRecurringDSTSpringForwardAndFallBack(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	// 2024-03-10 02:00–03:00 skipped. 02:30 snaps to the first valid clock (03:00 EDT).
	gap := CivilLocalTime(2024, 3, 10, 2, 30, ny)
	if gap.In(ny).Hour() != 3 || gap.In(ny).Minute() != 0 {
		t.Fatalf("spring-forward 02:30 → %v", gap.In(ny))
	}
	// 2024-11-03 01:30 happens twice; first occurrence is EDT (-04:00).
	dup := CivilLocalTime(2024, 11, 3, 1, 30, ny)
	_, off := dup.Zone()
	if off != -4*3600 {
		t.Fatalf("fall-back should pick first 01:30 (EDT), got %v off=%d", dup, off)
	}
	// Weekly Monday 09:00 keeps wall clock across the March 2024 spring-forward.
	p := RecurringBuyPlan{
		Frequency: RecurringWeekly, Weekday: "monday",
		TimeZone: "America/New_York", HasLocalTime: true, Hour: 9, Minute: 0,
	}
	before := time.Date(2024, 3, 4, 9, 0, 0, 0, ny) // EST
	next := AdvanceRecurringSchedule(before.UTC(), p)
	want := time.Date(2024, 3, 11, 9, 0, 0, 0, ny) // EDT
	if !next.Equal(want.UTC()) {
		t.Fatalf("DST week: got %v want %v (local %v)", next, want.UTC(), next.In(ny))
	}
	if before.UTC().Hour() == next.Hour() {
		t.Fatal("UTC hour should shift when local offset changes")
	}
}

func TestRecurringBudgetAndEndDate(t *testing.T) {
	if _, err := ResolveRecurringBudget(-1); err == nil {
		t.Fatal("negative budget")
	}
	if v, err := ResolveRecurringBudget(5000); err != nil || v != 5000 {
		t.Fatalf("%v %v", v, err)
	}
	p := RecurringBuyPlan{Amount: 500, Budget: 1200, Spent: 800}
	spend, reason := RecurringSpendAmount(p)
	if reason != "" || spend != 400 {
		t.Fatalf("partial leftover spend=%v reason=%q", spend, reason)
	}
	p.Spent = 1200
	if _, reason := RecurringSpendAmount(p); reason != "budget exhausted" {
		t.Fatalf("exhausted: %q", reason)
	}
	loc, _ := time.LoadLocation("America/New_York")
	end, err := ParseRecurringEndDate("2026-12-31", "America/New_York")
	if err != nil || end == nil {
		t.Fatal(err)
	}
	if end.In(loc).Format("2006-01-02") != "2026-12-31" {
		t.Fatalf("end local date %v", end.In(loc))
	}
	lastSlot := time.Date(2026, 12, 31, 9, 0, 0, 0, loc)
	plan := RecurringBuyPlan{EndsAt: end, TimeZone: "America/New_York"}
	if !RecurringPlanAllowsRun(plan, lastSlot) {
		t.Fatal("last local day should run")
	}
	if RecurringPlanAllowsRun(plan, lastSlot.AddDate(0, 0, 1)) {
		t.Fatal("day after end should not run")
	}
}

func TestRecurringDailyNineIstanbulAfterHour(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		t.Fatal(err)
	}
	// Monday 10:00 Istanbul → next is Tuesday 09:00
	from := time.Date(2024, 6, 10, 10, 0, 0, 0, loc)
	p := RecurringBuyPlan{
		Frequency: RecurringDaily, TimeZone: "Europe/Istanbul",
		HasLocalTime: true, Hour: 9, Minute: 0,
	}
	got := AlignRecurringStart(from, p)
	want := time.Date(2024, 6, 11, 9, 0, 0, 0, loc).UTC()
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

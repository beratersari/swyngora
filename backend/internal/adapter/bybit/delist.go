package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	announcementsPath = "/v5/announcements/index"
	// List descriptions are often empty; pair names and the halt clock live
	// on the announcement HTML page. Cap fetches so a refresh stays bounded.
	maxDelistArticleFetches = 25
	maxDelistArticleBytes   = 512 << 10
)

var (
	// May 19, 2026 / May 19, 2026, 9:00AM UTC / Aug 27, 2026, 8AM UTC
	delistDateLong = regexp.MustCompile(
		`(?i)\b(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2}),?\s+(\d{4})(?:\s*,?\s+(\d{1,2})(?::(\d{2}))?\s*(AM|PM)\s*UTC)?`,
	)
	delistDateShort = regexp.MustCompile(
		`(?i)\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{1,2}),?\s+(\d{4})(?:\s*,?\s+(\d{1,2})(?::(\d{2}))?\s*(AM|PM)\s*UTC)?`,
	)
	isoDate = regexp.MustCompile(`\b(20\d{2})-(\d{2})-(\d{2})(?:[ T](\d{1,2}):(\d{2})(?::(\d{2}))?\s*UTC)?`)
	// BTCUSDT / ETHUSDC / SOL-USDT / PEPE/USDT / VANRY/USDT / ETHMNT
	pairToken = regexp.MustCompile(`\b([A-Z0-9]{2,15})[-/]?(USDT|USDC|USDE|FDUSD|USD|EUR|BTC|ETH|MNT|DAI|TRY)\b`)
	scriptRe  = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRe   = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	tagRe     = regexp.MustCompile(`(?s)<[^>]+>`)
	spaceRe   = regexp.MustCompile(`\s+`)
)

var delistTimePhrases = []string{
	"will end after",
	"will end on",
	"end after",
	"end on",
	"trading pairs on",
	"spot trading pairs on",
	"pairs on",
}

type announcementsResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []announcementRow `json:"list"`
	} `json:"result"`
}

type announcementRow struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	URL           string `json:"url"`
	PublishTime   int64  `json:"publishTime"`
	DateTimestamp int64  `json:"dateTimestamp"`
	Type          struct {
		Key string `json:"key"`
	} `json:"type"`
	Tags []string `json:"tags"`
}

// FetchSpotDelistSchedule implements domain.SpotDelistSchedulePort from public
// Bybit announcements (no API key). Only rows with a parsed date + symbol.
func (c *Client) FetchSpotDelistSchedule(ctx context.Context) ([]domain.SpotDelistEntry, error) {
	params := url.Values{}
	params.Set("locale", "en-US")
	params.Set("type", "delistings")
	params.Set("limit", "50")
	body, err := c.get(ctx, announcementsPath, params)
	if err != nil {
		return nil, err
	}
	var resp announcementsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("%w: decode bybit announcements: %v", domain.ErrUpstream, err)
	}
	if resp.RetCode != 0 {
		return nil, mapBybitError(resp.RetCode, resp.RetMsg)
	}
	now := time.Now().UTC()
	cutoff := now.Add(-domain.DelistPastHorizon)
	out := make([]domain.SpotDelistEntry, 0, 16)
	seen := map[string]time.Time{}
	articleFetches := 0
	for _, row := range resp.Result.List {
		if !announcementLooksLikeSpotDelist(row) {
			continue
		}
		text := strings.TrimSpace(row.Title + "\n" + row.Description)
		when, hasTime := parseDelistTime(text)
		syms := parseDelistSymbols(text)
		// Older than 30 days is out of the Markets window; skip the HTML fetch.
		if hasTime && when.Before(cutoff) {
			continue
		}
		if (!hasTime || len(syms) == 0) && row.URL != "" && articleFetches < maxDelistArticleFetches {
			articleFetches++
			if article, aerr := c.fetchAnnouncementArticle(ctx, row.URL); aerr == nil && article != "" {
				text = text + "\n" + article
				when, hasTime = parseDelistTime(text)
				syms = parseDelistSymbols(text)
			}
		}
		if !hasTime || when.Before(cutoff) {
			continue
		}
		announced := unixMillisUTC(row.PublishTime)
		if announced.IsZero() {
			announced = unixMillisUTC(row.DateTimestamp)
		}
		for _, sym := range syms {
			if prev, ok := seen[sym]; ok && !when.Before(prev) {
				continue
			}
			seen[sym] = when
			out = append(out, domain.SpotDelistEntry{
				Exchange:    domain.ExchangeBybit,
				Symbol:      sym,
				DelistTime:  when,
				AnnouncedAt: announced,
			})
		}
	}
	return out, nil
}

func announcementLooksLikeSpotDelist(row announcementRow) bool {
	blob := strings.ToLower(row.Title + " " + row.Description + " " + strings.Join(row.Tags, " "))
	if !strings.Contains(blob, "delist") {
		return false
	}
	if strings.Contains(blob, "bybit alpha") {
		return false
	}
	if strings.Contains(blob, "as collateral") || strings.Contains(blob, "lending asset") {
		return false
	}
	if strings.Contains(blob, "perpetual") && !strings.Contains(blob, "spot") {
		return false
	}
	if strings.Contains(blob, "futures") && !strings.Contains(blob, "spot") {
		return false
	}
	return true
}

func parseDelistTime(text string) (time.Time, bool) {
	lower := strings.ToLower(text)
	for _, phrase := range delistTimePhrases {
		i := strings.Index(lower, phrase)
		if i < 0 {
			continue
		}
		window := text[i:]
		if len(window) > 96 {
			window = window[:96]
		}
		if t, ok := parseAnyDate(window); ok {
			return t, true
		}
	}
	return parseAnyDate(text)
}

func parseAnyDate(text string) (time.Time, bool) {
	if t, ok := parseNamedMonthDate(delistDateLong, text, false); ok {
		return t, true
	}
	if t, ok := parseNamedMonthDate(delistDateShort, text, true); ok {
		return t, true
	}
	if t, ok := parseISODate(text); ok {
		return t, true
	}
	return time.Time{}, false
}

func parseISODate(text string) (time.Time, bool) {
	m := isoDate.FindStringSubmatch(text)
	if len(m) < 4 {
		return time.Time{}, false
	}
	y, _ := strconv.Atoi(m[1])
	mo, _ := strconv.Atoi(m[2])
	d, _ := strconv.Atoi(m[3])
	hour, min, sec := 0, 0, 0
	if len(m) >= 6 && m[4] != "" {
		hour, _ = strconv.Atoi(m[4])
		min, _ = strconv.Atoi(m[5])
	}
	if len(m) >= 7 && m[6] != "" {
		sec, _ = strconv.Atoi(m[6])
	}
	return time.Date(y, time.Month(mo), d, hour, min, sec, 0, time.UTC), true
}

func parseNamedMonthDate(re *regexp.Regexp, text string, short bool) (time.Time, bool) {
	m := re.FindStringSubmatch(text)
	if len(m) < 4 {
		return time.Time{}, false
	}
	var month time.Month
	if short {
		month = shortMonth(m[1])
	} else {
		month = longMonth(m[1])
	}
	if month == 0 {
		return time.Time{}, false
	}
	day, _ := strconv.Atoi(m[2])
	year, _ := strconv.Atoi(m[3])
	hour, min := 0, 0
	if len(m) >= 7 && m[4] != "" {
		hour, _ = strconv.Atoi(m[4])
		if m[5] != "" {
			min, _ = strconv.Atoi(m[5])
		}
		if strings.EqualFold(m[6], "PM") && hour < 12 {
			hour += 12
		}
		if strings.EqualFold(m[6], "AM") && hour == 12 {
			hour = 0
		}
	}
	return time.Date(year, month, day, hour, min, 0, 0, time.UTC), true
}

func (c *Client) fetchAnnouncementArticle(ctx context.Context, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("invalid announcement url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "swyngora-backend/0.1")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("announcement status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDelistArticleBytes))
	if err != nil {
		return "", err
	}
	return stripHTML(string(body)), nil
}

func stripHTML(s string) string {
	s = scriptRe.ReplaceAllString(s, " ")
	s = styleRe.ReplaceAllString(s, " ")
	s = tagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func unixMillisUTC(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	if ms < 1e12 {
		return time.Unix(ms, 0).UTC()
	}
	return time.UnixMilli(ms).UTC()
}

func parseDelistSymbols(text string) []string {
	up := strings.ToUpper(text)
	found := map[string]struct{}{}
	var out []string
	for _, m := range pairToken.FindAllStringSubmatch(up, -1) {
		if len(m) < 3 {
			continue
		}
		base, quote := m[1], m[2]
		if base == "" || quote == "" {
			continue
		}
		sym := base + quote
		if _, ok := found[sym]; ok {
			continue
		}
		found[sym] = struct{}{}
		out = append(out, sym)
	}
	return out
}

func longMonth(s string) time.Month {
	switch strings.ToLower(s) {
	case "january":
		return time.January
	case "february":
		return time.February
	case "march":
		return time.March
	case "april":
		return time.April
	case "may":
		return time.May
	case "june":
		return time.June
	case "july":
		return time.July
	case "august":
		return time.August
	case "september":
		return time.September
	case "october":
		return time.October
	case "november":
		return time.November
	case "december":
		return time.December
	default:
		return 0
	}
}

func shortMonth(s string) time.Month {
	switch strings.ToLower(s) {
	case "jan":
		return time.January
	case "feb":
		return time.February
	case "mar":
		return time.March
	case "apr":
		return time.April
	case "may":
		return time.May
	case "jun":
		return time.June
	case "jul":
		return time.July
	case "aug":
		return time.August
	case "sep":
		return time.September
	case "oct":
		return time.October
	case "nov":
		return time.November
	case "dec":
		return time.December
	default:
		return 0
	}
}

package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/pkg/lokiclient"
)

// systemLogAppLabel is the constant Loki label distinguishing this
// application's streams (data-model.md). category is the only other label —
// actor/target are deliberately NOT labels (high-cardinality — research.md
// Decision 1); they live in the JSON line body instead.
const systemLogAppLabel = "mansooba"

// maxFetchEntries bounds how many matching entries a single FindPaginated
// call scans to compute Total and slice out the requested page. Loki has no
// cheap exact-count query; this fetches a generous, bounded window instead
// (contracts/system-logs-api.md: "total is Loki's returned match count... an
// estimate, not a guarantee") — acceptable at this project's expected scale
// (90-day default retention, four event categories only).
const maxFetchEntries = 1000

// defaultLookback bounds how far back an unfiltered query (no From) reaches.
// Loki rejects any query_range wider than limits_config.max_query_length
// (loki/local-config.yaml sets 8760h/365d) — time.Unix(0,0) as a "beginning
// of time" sentinel produced a ~56-year range and every unfiltered query
// 500'd. This must stay <= that config value; it's set equal to it so "no
// date filter" means "everything Loki will actually search," matching
// FR-006's "no filter = all retained entries" intent as closely as Loki's
// query-length limit allows.
const defaultLookback = 365 * 24 * time.Hour

// logLine is the JSON shape pushed to and parsed from Loki's line body —
// everything in a SystemLogEntry except Category (a label) and CreatedAt
// (the line's own Loki timestamp).
type logLine struct {
	Action   string `json:"action"`
	Outcome  string `json:"outcome"`
	Actor    string `json:"actor"`
	ActorID  uint   `json:"actor_id,omitempty"`
	Target   string `json:"target,omitempty"`
	TargetID uint   `json:"target_id,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type lokiSystemLogRepository struct {
	client *lokiclient.Client
}

// NewLokiSystemLogRepository returns a domain.SystemLogRepository backed by
// Grafana Loki (ADR-031) — no GORM/database involvement at all.
func NewLokiSystemLogRepository(client *lokiclient.Client) domain.SystemLogRepository {
	return &lokiSystemLogRepository{client: client}
}

func (r *lokiSystemLogRepository) Create(ctx context.Context, entry domain.SystemLogEntry) error {
	line, err := json.Marshal(logLine{
		Action:   entry.Action,
		Outcome:  entry.Outcome,
		Actor:    entry.Actor,
		ActorID:  entry.ActorID,
		Target:   entry.Target,
		TargetID: entry.TargetID,
		Detail:   entry.Detail,
	})
	if err != nil {
		return fmt.Errorf("loki_systemlog_repository: marshal line: %w", err)
	}

	ts := entry.CreatedAt
	if ts.IsZero() {
		ts = time.Now()
	}

	labels := map[string]string{"app": systemLogAppLabel}
	if entry.Category != "" {
		labels["category"] = entry.Category
	}

	return r.client.Push(ctx, labels, ts, string(line))
}

func (r *lokiSystemLogRepository) FindPaginated(ctx context.Context, filter domain.SystemLogListFilter) (domain.SystemLogListResult, error) {
	query := buildLogQL(filter)

	end := time.Now()
	start := end.Add(-defaultLookback)
	if filter.From != nil {
		start = *filter.From
	}
	if filter.To != nil {
		end = *filter.To
	}

	entries, err := r.client.QueryRange(ctx, lokiclient.QueryRangeParams{
		Query:     query,
		Start:     start,
		End:       end,
		Limit:     maxFetchEntries,
		Direction: "backward",
	})
	if err != nil {
		return domain.SystemLogListResult{}, fmt.Errorf("loki_systemlog_repository: query_range: %w", err)
	}

	// QueryRange doesn't guarantee cross-stream ordering (each label-selected
	// stream is individually newest-first); re-sort the combined set to
	// guarantee a single newest-first order across categories.
	sort.Slice(entries, func(i, j int) bool { return entries[i].Timestamp.After(entries[j].Timestamp) })

	all := make([]domain.SystemLogEntry, 0, len(entries))
	for _, e := range entries {
		all = append(all, mapLogEntry(e))
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	size := filter.Size
	if size < 1 {
		size = 20
	}

	total := len(all)
	from := (page - 1) * size
	if from > total {
		from = total
	}
	to := from + size
	if to > total {
		to = total
	}

	return domain.SystemLogListResult{
		Entries: all[from:to],
		Total:   total,
	}, nil
}

// buildLogQL translates filter into a LogQL query string. category becomes a
// label selector; actor becomes a `| json` field filter (data-model.md —
// distinct from q's broader raw line filter); q is a case-insensitive line
// filter over the entire JSON body. User-supplied text is always passed
// through regexp.QuoteMeta so it behaves as a literal substring match, never
// as attacker-controlled regex.
func buildLogQL(filter domain.SystemLogListFilter) string {
	selector := fmt.Sprintf(`{app=%q`, systemLogAppLabel)
	if filter.Category != "" {
		selector += fmt.Sprintf(`,category=%q`, filter.Category)
	}
	selector += "}"

	var parts []string
	parts = append(parts, selector)

	if filter.Q != "" {
		parts = append(parts, fmt.Sprintf(`|~ %s`, logQLPattern(filter.Q)))
	}
	if filter.Actor != "" {
		parts = append(parts, "| json", fmt.Sprintf(`| actor=~ %s`, logQLPattern(filter.Actor)))
	}

	return strings.Join(parts, " ")
}

// logQLPattern turns raw user text into a quoted, case-insensitive LogQL
// regex literal matching it verbatim. LogQL string literals use the same
// backslash-escaping as Go string literals — regexp.QuoteMeta backslash-
// escapes regex metacharacters (e.g. "." -> "\."), but embedding that
// directly in a LogQL "..." literal makes the *string* parser consume the
// single backslash as an (invalid) escape sequence before the regex engine
// ever sees it, so any query containing a metacharacter — a "." in an email
// address, the single most common case — 400'd with "invalid char escape".
// %q re-escapes the already-QuoteMeta'd pattern for the LogQL string layer
// (doubling backslashes, escaping quotes), matching what a hand-written
// LogQL query would need.
func logQLPattern(raw string) string {
	return fmt.Sprintf("%q", "(?i)"+regexp.QuoteMeta(raw))
}

func mapLogEntry(e lokiclient.LogEntry) domain.SystemLogEntry {
	var line logLine
	_ = json.Unmarshal([]byte(e.Line), &line) // best-effort — a malformed line still shows up with empty fields rather than being dropped

	return domain.SystemLogEntry{
		ID:        synthesizeID(e.Timestamp, e.Line),
		Category:  e.Labels["category"],
		Action:    line.Action,
		Outcome:   line.Outcome,
		Actor:     line.Actor,
		ActorID:   line.ActorID,
		Target:    line.Target,
		TargetID:  line.TargetID,
		Detail:    line.Detail,
		CreatedAt: e.Timestamp,
	}
}

// synthesizeID builds an opaque, stable-enough-for-:key ID from the entry's
// timestamp and a short content hash (contracts/system-logs-api.md) — Loki
// has no row ID, unlike the GORM auto-increment column the original design
// would have had.
func synthesizeID(ts time.Time, line string) string {
	sum := sha256.Sum256([]byte(line))
	return fmt.Sprintf("%d-%s", ts.UnixNano(), hex.EncodeToString(sum[:])[:6])
}

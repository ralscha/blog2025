package demo

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	store "paradedb-demo/internal/store/sqlc"
)

// Options configures the demo runner.
type Options struct {
	DB    *sql.DB
	Limit int
}

type table struct {
	headers []string
	rows    [][]string
}

// Run executes a series of showcase queries that mirror the examples in the
// blog post and prints formatted results to stdout. Errors from individual
// sections are printed but do not stop subsequent sections from running.
func Run(ctx context.Context, opts Options) error {
	if opts.Limit <= 0 {
		opts.Limit = 5
	}
	queries := store.New(opts.DB)
	limit := int32(opts.Limit)

	type section struct {
		title string
		run   func(context.Context, *store.Queries, int32) (table, error)
	}

	sections := []section{
		{
			title: "Match Disjunction (|||) — content OR-matches 'love marriage'",
			run: func(ctx context.Context, queries *store.Queries, limit int32) (table, error) {
				rows, err := queries.DemoMatchDisjunction(ctx, limit)
				return tableFromMatchRows(rows), err
			},
		},
		{
			title: "Match Conjunction (&&&) — content AND-matches 'love marriage'",
			run: func(ctx context.Context, queries *store.Queries, limit int32) (table, error) {
				rows, err := queries.DemoMatchConjunction(ctx, limit)
				return tableFromMatchConjunctionRows(rows), err
			},
		},
		{
			title: "Phrase (###) — content contains exact phrase 'single man'",
			run: func(ctx context.Context, queries *store.Queries, limit int32) (table, error) {
				rows, err := queries.DemoPhrase(ctx, limit)
				return tableFromPhraseRows(rows), err
			},
		},
		{
			title: "Phrase with Slop (###::pdb.slop) — 'marriage love' within 2 positional changes",
			run: func(ctx context.Context, queries *store.Queries, limit int32) (table, error) {
				rows, err := queries.DemoPhraseWithSlop(ctx, limit)
				return tableFromPhraseWithSlopRows(rows), err
			},
		},
		{
			title: "Term (===) — exact token match on language field",
			run: func(ctx context.Context, queries *store.Queries, limit int32) (table, error) {
				rows, err := queries.DemoTermLanguage(ctx, limit)
				return tableFromLanguageRows(rows), err
			},
		},
		{
			title: "Fuzzy — 'marrige' (typo) matches 'marriage' within edit distance 2",
			run: func(ctx context.Context, queries *store.Queries, limit int32) (table, error) {
				rows, err := queries.DemoFuzzy(ctx, limit)
				return tableFromFuzzyRows(rows), err
			},
		},
		{
			title: "Highlighting — pdb.snippet returns matched excerpt with <b> tags",
			run: func(ctx context.Context, queries *store.Queries, limit int32) (table, error) {
				rows, err := queries.DemoHighlighting(ctx, limit)
				return tableFromHighlightRows(rows), err
			},
		},
		{
			title: "Proximity (##) — 'love' within 5 tokens of 'marriage'",
			run: func(ctx context.Context, queries *store.Queries, limit int32) (table, error) {
				rows, err := queries.DemoProximity(ctx, limit)
				return tableFromProximityRows(rows), err
			},
		},
		{
			title: "Filtering — BM25 search combined with SQL predicate (download_count > 100)",
			run: func(ctx context.Context, queries *store.Queries, limit int32) (table, error) {
				rows, err := queries.DemoFiltering(ctx, limit)
				return tableFromFilteringRows(rows), err
			},
		},
		{
			title: "Boosting — title matches weighted 3× over content matches",
			run: func(ctx context.Context, queries *store.Queries, limit int32) (table, error) {
				rows, err := queries.DemoBoosting(ctx, limit)
				return tableFromBoostingRows(rows), err
			},
		},
		{
			title: "Aggregate Metrics — avg and total count via pdb.agg",
			run: func(ctx context.Context, queries *store.Queries, _ int32) (table, error) {
				row, err := queries.DemoAggregateMetrics(ctx)
				if err != nil {
					return table{}, err
				}
				return table{
					headers: []string{"avg_downloads", "total_books"},
					rows:    [][]string{{row.AvgDownloads, row.TotalBooks}},
				}, nil
			},
		},
		{
			title: "Facets — Top K results with aggregate total in a single query",
			run: func(ctx context.Context, queries *store.Queries, _ int32) (table, error) {
				rows, err := queries.DemoFacets(ctx)
				return tableFromFacetRows(rows), err
			},
		},
		{
			title: "Advanced Query — regex match on author tokens",
			run: func(ctx context.Context, queries *store.Queries, limit int32) (table, error) {
				rows, err := queries.DemoRegex(ctx, limit)
				return tableFromRegexRows(rows), err
			},
		},
		{
			title: "Advanced Query — boolean must/must_not",
			run: func(ctx context.Context, queries *store.Queries, limit int32) (table, error) {
				rows, err := queries.DemoBoolean(ctx, limit)
				return tableFromBooleanRows(rows), err
			},
		},
		{
			title: "Advanced Query — standard SQL range filter on download_count",
			run: func(ctx context.Context, queries *store.Queries, limit int32) (table, error) {
				rows, err := queries.DemoRange(ctx, limit)
				return tableFromRangeRows(rows), err
			},
		},
	}

	for _, s := range sections {
		tbl, err := s.run(ctx, queries, limit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR: %v\n\n", err)
			continue
		}
		printSection(s.title, tbl)
	}
	return nil
}

func printSection(title string, tbl table) {
	const lineWidth = 76
	sep := repeatRune('─', lineWidth)

	fmt.Printf("\n%s\n%s\n%s\n\n", sep, title, sep)

	w := tabwriter.NewWriter(os.Stdout, 1, 0, 2, ' ', 0)
	fmt.Fprintln(w, joinTabs(tbl.headers))

	dashes := make([]string, len(tbl.headers))
	for i, c := range tbl.headers {
		dashes[i] = strings.Repeat("-", len(c))
	}
	fmt.Fprintln(w, joinTabs(dashes))

	for _, row := range tbl.rows {
		fmt.Fprintln(w, joinTabs(row))
	}
	_ = w.Flush()

	if len(tbl.rows) == 0 {
		fmt.Println("  (no rows)")
	} else {
		fmt.Printf("  %d row(s)\n", len(tbl.rows))
	}
	fmt.Println()
}

func tableFromMatchRows(rows []store.DemoMatchDisjunctionRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "author", "score"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, row.Author, row.Score})
	}
	return result
}

func tableFromMatchConjunctionRows(rows []store.DemoMatchConjunctionRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "author", "score"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, row.Author, row.Score})
	}
	return result
}

func tableFromPhraseRows(rows []store.DemoPhraseRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "author", "score"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, row.Author, row.Score})
	}
	return result
}

func tableFromPhraseWithSlopRows(rows []store.DemoPhraseWithSlopRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "author", "score"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, row.Author, row.Score})
	}
	return result
}

func tableFromLanguageRows(rows []store.DemoTermLanguageRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "author", "language"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, row.Author, row.Language})
	}
	return result
}

func tableFromFuzzyRows(rows []store.DemoFuzzyRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "author", "score"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, row.Author, row.Score})
	}
	return result
}

func tableFromHighlightRows(rows []store.DemoHighlightingRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "snippet"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, formatVal(row.Snippet)})
	}
	return result
}

func tableFromProximityRows(rows []store.DemoProximityRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "author", "score"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, row.Author, row.Score})
	}
	return result
}

func tableFromFilteringRows(rows []store.DemoFilteringRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "author", "download_count", "score"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, row.Author, row.DownloadCount, row.Score})
	}
	return result
}

func tableFromBoostingRows(rows []store.DemoBoostingRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "author", "score"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, row.Author, row.Score})
	}
	return result
}

func tableFromFacetRows(rows []store.DemoFacetsRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "author", "total_matches"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, row.Author, row.TotalMatches})
	}
	return result
}

func tableFromRegexRows(rows []store.DemoRegexRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "author"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, row.Author})
	}
	return result
}

func tableFromBooleanRows(rows []store.DemoBooleanRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "author", "score"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, row.Author, row.Score})
	}
	return result
}

func tableFromRangeRows(rows []store.DemoRangeRow) table {
	result := table{headers: []string{"gutenberg_id", "title", "author", "download_count"}, rows: make([][]string, 0, len(rows))}
	for _, row := range rows {
		result.rows = append(result.rows, []string{row.GutenbergID, row.Title, row.Author, row.DownloadCount})
	}
	return result
}

func joinTabs(values []string) string {
	return strings.Join(values, "\t")
}

func repeatRune(value rune, count int) string {
	if count <= 0 {
		return ""
	}

	buf := make([]rune, count)
	for i := range buf {
		buf[i] = value
	}
	return string(buf)
}

// formatVal converts a scanned database value to a display string.
func formatVal(v any) string {
	if v == nil {
		return "NULL"
	}
	const maxLen = 100
	truncate := func(s string) string {
		if len(s) > maxLen {
			return s[:maxLen-3] + "..."
		}
		return s
	}
	switch t := v.(type) {
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int32:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case float32:
		return fmt.Sprintf("%.4f", t)
	case float64:
		return fmt.Sprintf("%.4f", t)
	case []byte:
		return truncate(string(t))
	case string:
		return truncate(t)
	default:
		return truncate(fmt.Sprintf("%v", t))
	}
}

package search

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	store "paradedb-demo/internal/store/sqlc"
)

type Options struct {
	DB          *sql.DB
	Query       string
	Limit       int
	Author      string
	Language    string
	Snippet     bool
	Conjunction bool
}

type Result struct {
	GutenbergID   int64
	Title         string
	Author        string
	Language      string
	DownloadCount int
	ContentBytes  int64
	Score         float64
	Snippet       string
}

func Run(ctx context.Context, opts Options) ([]Result, error) {
	if opts.DB == nil {
		return nil, fmt.Errorf("search requires a database handle")
	}

	queryText := strings.TrimSpace(opts.Query)
	if queryText == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	authorPattern := ""
	if author := strings.TrimSpace(opts.Author); author != "" {
		authorPattern = "%" + author + "%"
	}

	queries := store.New(opts.DB)
	rows, err := queries.SearchBooks(ctx, store.SearchBooksParams{
		QueryText:      queryText,
		UseConjunction: opts.Conjunction,
		IncludeSnippet: opts.Snippet,
		AuthorPattern:  authorPattern,
		LanguageFilter: strings.TrimSpace(opts.Language),
		LimitRows:      int32(opts.Limit),
	})
	if err != nil {
		return nil, fmt.Errorf("query books: %w", err)
	}

	results := make([]Result, 0, len(rows))
	for _, row := range rows {
		results = append(results, Result{
			GutenbergID:   row.GutenbergID,
			Title:         row.Title,
			Author:        row.Author,
			Language:      row.Language,
			DownloadCount: int(row.DownloadCount),
			ContentBytes:  row.ContentBytes,
			Score:         row.Score,
			Snippet:       row.Snippet,
		})
	}

	return results, nil
}

func PrintResults(results []Result) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "ID\tSCORE\tLANG\tDOWNLOADS\tMB\tAUTHOR\tTITLE")
	for _, result := range results {
		_, _ = fmt.Fprintf(
			writer,
			"%d\t%.4f\t%s\t%d\t%.2f\t%s\t%s\n",
			result.GutenbergID,
			result.Score,
			result.Language,
			result.DownloadCount,
			float64(result.ContentBytes)/(1024*1024),
			result.Author,
			result.Title,
		)
	}
	_ = writer.Flush()

	for _, result := range results {
		if result.Snippet == "" {
			continue
		}
		_, _ = fmt.Fprintf(os.Stdout, "\n[%d] %s\n%s\n", result.GutenbergID, result.Title, result.Snippet)
	}
}

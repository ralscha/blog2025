package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"paradedb-demo/internal/config"
	appdb "paradedb-demo/internal/db"
	"paradedb-demo/internal/demo"
	"paradedb-demo/internal/importer"
	"paradedb-demo/internal/search"
)

func closeDB(db *sql.DB, errp *error) {
	if err := db.Close(); err != nil && *errp == nil {
		*errp = err
	}
}

func openDatabase(ctx context.Context, databaseURL string) (*sql.DB, error) {
	dbConn, err := appdb.Open(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	if err := appdb.RunMigrations(ctx, dbConn); err != nil {
		_ = dbConn.Close()
		return nil, err
	}

	return dbConn, nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "import":
		err = runImport(ctx, os.Args[2:])
	case "search":
		err = runSearch(ctx, os.Args[2:])
	case "demo":
		err = runDemo(ctx, os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		usage()
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runImport(ctx context.Context, args []string) (retErr error) {
	flags := flag.NewFlagSet("import", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)

	dbURL := flags.String("db-url", config.DatabaseURL(), "Postgres connection string")
	dataDir := flags.String("data-dir", config.DownloadsDir(), "Directory used to cache downloaded book files")
	topicsFile := flags.String("topics-file", config.TopicsPath(), "Newline-delimited list of Gutenberg topics to crawl")
	targetGB := flags.String("target-gb", "1.2", "Target corpus size in GiB before the importer stops downloading")
	maxBooks := flags.Int("max-books", 500, "Maximum number of books to import")
	pagesPerTopic := flags.Int("pages-per-topic", 8, "How many Gutendex result pages to fetch per topic")
	minBookKB := flags.Int64("min-book-kb", 64, "Skip books whose normalized content is smaller than this many KiB")
	language := flags.String("language", "en", "Language filter passed to Gutendex")

	if err := flags.Parse(args); err != nil {
		return err
	}

	targetBytes, err := importer.ParseTargetGB(*targetGB)
	if err != nil {
		return err
	}

	dbConn, err := openDatabase(ctx, *dbURL)
	if err != nil {
		return err
	}
	defer closeDB(dbConn, &retErr)

	started := time.Now()
	result, err := importer.Run(ctx, importer.Options{
		DB:            dbConn,
		DataDir:       *dataDir,
		TopicsFile:    *topicsFile,
		TargetBytes:   targetBytes,
		MaxBooks:      *maxBooks,
		PagesPerTopic: *pagesPerTopic,
		MinBookBytes:  *minBookKB * 1024,
		Language:      *language,
		Stdout:        os.Stdout,
	})
	if err != nil {
		return err
	}

	fmt.Printf("\nImported %d books, downloaded %d new files, total normalized corpus %.2f GiB in %s.\n", result.ImportedBooks, result.DownloadedBooks, float64(result.ImportedBytes)/(1024*1024*1024), time.Since(started).Round(time.Second))
	return nil
}

func runSearch(ctx context.Context, args []string) (retErr error) {
	flags := flag.NewFlagSet("search", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)

	dbURL := flags.String("db-url", config.DatabaseURL(), "Postgres connection string")
	queryFlag := flags.String("query", "", "Search text. If omitted, trailing positional arguments are joined into the query.")
	limit := flags.Int("limit", 10, "Maximum number of results to return")
	author := flags.String("author", "", "Optional author substring filter")
	language := flags.String("language", "", "Optional exact language filter")
	snippet := flags.Bool("snippet", true, "Whether to include ParadeDB snippets in the output")
	conjunction := flags.Bool("and", false, "Use ParadeDB conjunction matching (&&&) instead of disjunction (|||)")

	if err := flags.Parse(args); err != nil {
		return err
	}

	query := strings.TrimSpace(*queryFlag)
	if query == "" {
		query = strings.TrimSpace(strings.Join(flags.Args(), " "))
	}
	if query == "" {
		return fmt.Errorf("search requires a query string")
	}

	dbConn, err := openDatabase(ctx, *dbURL)
	if err != nil {
		return err
	}
	defer closeDB(dbConn, &retErr)

	results, err := search.Run(ctx, search.Options{
		DB:          dbConn,
		Query:       query,
		Limit:       *limit,
		Author:      *author,
		Language:    *language,
		Snippet:     *snippet,
		Conjunction: *conjunction,
	})
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Println("No results matched the query.")
		return nil
	}

	search.PrintResults(results)
	return nil
}

func runDemo(ctx context.Context, args []string) (retErr error) {
	flags := flag.NewFlagSet("demo", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)

	dbURL := flags.String("db-url", config.DatabaseURL(), "Postgres connection string")
	limit := flags.Int("limit", 5, "Maximum rows to display per query")

	if err := flags.Parse(args); err != nil {
		return err
	}

	dbConn, err := openDatabase(ctx, *dbURL)
	if err != nil {
		return err
	}
	defer closeDB(dbConn, &retErr)

	return demo.Run(ctx, demo.Options{
		DB:    dbConn,
		Limit: *limit,
	})
}

func usage() {
	fmt.Print(`paradedb-demo is a small CLI for building a ParadeDB book-search demo.

Commands:
  import    Discover a Gutenberg corpus, download books, and upsert them into ParadeDB.
  search    Run a ParadeDB BM25 query with optional filters and snippets.
  demo      Run all blog-post showcase queries and print formatted results.

Examples:
  go run ./cmd/paradedb-demo import --target-gb 1.2 --max-books 500
  go run ./cmd/paradedb-demo import --topics-file data/manifest/gutenberg-smoke.txt --target-gb 0.05 --max-books 10 --pages-per-topic 1
  go run ./cmd/paradedb-demo search --limit 5 "white whale revenge"
  go run ./cmd/paradedb-demo search --and --author Austen "marriage pride"
  go run ./cmd/paradedb-demo demo --limit 5
`)
}

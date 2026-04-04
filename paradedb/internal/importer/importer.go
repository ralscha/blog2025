package importer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	store "paradedb-demo/internal/store/sqlc"
)

const gutendexBaseURL = "https://gutendex.com/books"

type Options struct {
	DB            *sql.DB
	DataDir       string
	TopicsFile    string
	TargetBytes   int64
	MaxBooks      int
	PagesPerTopic int
	MinBookBytes  int64
	Language      string
	HTTPClient    *http.Client
	Stdout        io.Writer
}

type gutendexResponse struct {
	Next    string         `json:"next"`
	Results []gutendexBook `json:"results"`
}

type gutendexBook struct {
	ID            int               `json:"id"`
	Title         string            `json:"title"`
	Subjects      []string          `json:"subjects"`
	Authors       []gutendexPerson  `json:"authors"`
	Bookshelves   []string          `json:"bookshelves"`
	Languages     []string          `json:"languages"`
	Copyright     *bool             `json:"copyright"`
	Formats       map[string]string `json:"formats"`
	DownloadCount int               `json:"download_count"`
}

type gutendexPerson struct {
	Name string `json:"name"`
}

type candidateBook struct {
	ID            int
	Title         string
	Author        string
	Language      string
	Subjects      []string
	Bookshelves   []string
	DownloadCount int
	SourceURL     string
	TextURL       string
}

type Result struct {
	ImportedBooks   int
	ImportedBytes   int64
	DownloadedBooks int
}

func Run(ctx context.Context, opts Options) (Result, error) {
	if opts.DB == nil {
		return Result{}, errors.New("importer requires a database handle")
	}
	if opts.DataDir == "" {
		return Result{}, errors.New("importer requires a data directory")
	}
	if opts.TopicsFile == "" {
		return Result{}, errors.New("importer requires a topics file")
	}
	if opts.TargetBytes <= 0 {
		return Result{}, errors.New("target bytes must be greater than zero")
	}
	if opts.MaxBooks <= 0 {
		return Result{}, errors.New("max books must be greater than zero")
	}
	if opts.PagesPerTopic <= 0 {
		opts.PagesPerTopic = 8
	}
	if opts.MinBookBytes <= 0 {
		opts.MinBookBytes = 50_000
	}
	if opts.Language == "" {
		opts.Language = "en"
	}
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 90 * time.Second}
	}

	queries := store.New(opts.DB)

	if err := os.MkdirAll(opts.DataDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create data dir: %w", err)
	}

	topics, err := loadTopics(opts.TopicsFile)
	if err != nil {
		return Result{}, err
	}

	candidates, err := discoverCandidates(ctx, opts.HTTPClient, topics, opts)
	if err != nil {
		return Result{}, err
	}
	if len(candidates) == 0 {
		return Result{}, errors.New("no Gutenberg books matched the configured corpus profile")
	}

	if err := writef(opts.Stdout, "Discovered %d candidate books across %d topics.\n", len(candidates), len(topics)); err != nil {
		return Result{}, fmt.Errorf("write discovery summary: %w", err)
	}

	var result Result
	for _, book := range candidates {
		if result.ImportedBooks >= opts.MaxBooks || result.ImportedBytes >= opts.TargetBytes {
			break
		}

		localPath := filepath.Join(opts.DataDir, fmt.Sprintf("%d.txt", book.ID))
		downloaded, size, err := ensureBookFile(ctx, opts.HTTPClient, book, localPath)
		if err != nil {
			if writeErr := writef(opts.Stdout, "Skipping %d (%s): %v\n", book.ID, book.Title, err); writeErr != nil {
				return result, fmt.Errorf("write skip message: %w", writeErr)
			}
			continue
		}
		if size < opts.MinBookBytes {
			if err := writef(opts.Stdout, "Skipping %d (%s): file too small after download (%d bytes).\n", book.ID, book.Title, size); err != nil {
				return result, fmt.Errorf("write size skip message: %w", err)
			}
			continue
		}

		contentBytes, content, err := loadBookContent(localPath)
		if err != nil {
			return result, fmt.Errorf("load book %d: %w", book.ID, err)
		}
		if int64(contentBytes) < opts.MinBookBytes {
			if err := writef(opts.Stdout, "Skipping %d (%s): normalized content too small (%d bytes).\n", book.ID, book.Title, contentBytes); err != nil {
				return result, fmt.Errorf("write normalized-size skip message: %w", err)
			}
			continue
		}

		if err := upsertBook(ctx, queries, book, content, int64(contentBytes)); err != nil {
			return result, fmt.Errorf("upsert book %d: %w", book.ID, err)
		}

		if downloaded {
			result.DownloadedBooks++
		}
		result.ImportedBooks++
		result.ImportedBytes += int64(contentBytes)

		if err := writef(opts.Stdout, "Imported %4d | %-60s | %8d bytes | total %.2f GB\n", book.ID, truncate(book.Title, 60), contentBytes, bytesToGigabytes(result.ImportedBytes)); err != nil {
			return result, fmt.Errorf("write import progress: %w", err)
		}
	}

	if err := queries.AnalyzeBooks(ctx); err != nil {
		return result, fmt.Errorf("analyze books: %w", err)
	}

	return result, nil
}

func loadTopics(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read topics file: %w", err)
	}

	var topics []string
	for line := range strings.SplitSeq(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		topics = append(topics, line)
	}

	if len(topics) == 0 {
		return nil, errors.New("topics file was empty")
	}

	return topics, nil
}

func discoverCandidates(ctx context.Context, client *http.Client, topics []string, opts Options) ([]candidateBook, error) {
	byID := make(map[int]candidateBook)

	for _, topic := range topics {
		if err := writef(opts.Stdout, "Discovering topic %q...\n", topic); err != nil {
			return nil, fmt.Errorf("write topic discovery message: %w", err)
		}
		nextURL := buildTopicURL(topic, opts.Language)
		for page := 0; page < opts.PagesPerTopic && nextURL != ""; page++ {
			if err := writef(opts.Stdout, "  Fetching page %d for %q\n", page+1, topic); err != nil {
				return nil, fmt.Errorf("write page fetch message: %w", err)
			}
			response, err := fetchBooksPage(ctx, client, nextURL)
			if err != nil {
				return nil, fmt.Errorf("fetch topic %q page %d: %w", topic, page+1, err)
			}

			for _, book := range response.Results {
				if book.Copyright != nil && *book.Copyright {
					continue
				}

				textURL := pickTextURL(book.Formats)
				if textURL == "" {
					continue
				}

				candidate := candidateBook{
					ID:            book.ID,
					Title:         strings.TrimSpace(book.Title),
					Author:        joinAuthors(book.Authors),
					Language:      firstLanguage(book.Languages, opts.Language),
					Subjects:      cleanStringSlice(book.Subjects),
					Bookshelves:   cleanStringSlice(book.Bookshelves),
					DownloadCount: book.DownloadCount,
					SourceURL:     fmt.Sprintf("https://www.gutenberg.org/ebooks/%d", book.ID),
					TextURL:       textURL,
				}

				if existing, ok := byID[book.ID]; ok && existing.DownloadCount >= candidate.DownloadCount {
					continue
				}
				byID[book.ID] = candidate
			}

			nextURL = response.Next
		}
		if err := writef(opts.Stdout, "Completed topic %q. Current candidate count: %d\n", topic, len(byID)); err != nil {
			return nil, fmt.Errorf("write topic completion message: %w", err)
		}
	}

	candidates := make([]candidateBook, 0, len(byID))
	for _, candidate := range byID {
		candidates = append(candidates, candidate)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].DownloadCount == candidates[j].DownloadCount {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].DownloadCount > candidates[j].DownloadCount
	})

	return candidates, nil
}

func buildTopicURL(topic, language string) string {
	values := url.Values{}
	values.Set("languages", language)
	values.Set("copyright", "false")
	values.Set("mime_type", "text/plain")
	values.Set("sort", "popular")
	values.Set("topic", topic)
	return gutendexBaseURL + "?" + values.Encode()
}

func fetchBooksPage(ctx context.Context, client *http.Client, endpoint string) (gutendexResponse, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		payload, err := fetchBooksPageOnce(ctx, client, endpoint)
		if err == nil {
			return payload, nil
		}

		lastErr = err
		if attempt == 3 || !isRetryableFetchError(err) {
			break
		}

		select {
		case <-ctx.Done():
			return gutendexResponse{}, ctx.Err()
		case <-time.After(time.Duration(attempt) * 2 * time.Second):
		}
	}

	return gutendexResponse{}, lastErr
}

func fetchBooksPageOnce(ctx context.Context, client *http.Client, endpoint string) (payload gutendexResponse, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return gutendexResponse{}, fmt.Errorf("build request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return gutendexResponse{}, fmt.Errorf("request page: %w", err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", closeErr)
		}
	}()

	if response.StatusCode != http.StatusOK {
		return gutendexResponse{}, fmt.Errorf("unexpected status %s", response.Status)
	}

	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return gutendexResponse{}, fmt.Errorf("decode response: %w", err)
	}

	return payload, nil
}

func isRetryableFetchError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "context deadline exceeded") ||
		strings.Contains(message, "timeout") ||
		strings.Contains(message, "unexpected status 429") ||
		strings.Contains(message, "unexpected status 500") ||
		strings.Contains(message, "unexpected status 502") ||
		strings.Contains(message, "unexpected status 503") ||
		strings.Contains(message, "unexpected status 504")
}

func pickTextURL(formats map[string]string) string {
	priorities := []string{
		"text/plain; charset=utf-8",
		"text/plain; charset=us-ascii",
		"text/plain",
		"text/plain; charset=iso-8859-1",
	}

	for _, key := range priorities {
		if value := strings.TrimSpace(formats[key]); value != "" {
			return value
		}
	}

	for key, value := range formats {
		if strings.HasPrefix(strings.ToLower(key), "text/plain") && strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

func joinAuthors(authors []gutendexPerson) string {
	if len(authors) == 0 {
		return "Unknown"
	}

	names := make([]string, 0, len(authors))
	for _, author := range authors {
		name := strings.TrimSpace(author.Name)
		if name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "Unknown"
	}

	return strings.Join(names, "; ")
}

func firstLanguage(languages []string, fallback string) string {
	if len(languages) == 0 {
		return fallback
	}

	for _, language := range languages {
		language = strings.TrimSpace(language)
		if language != "" {
			return language
		}
	}

	return fallback
}

func cleanStringSlice(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func ensureBookFile(ctx context.Context, client *http.Client, book candidateBook, localPath string) (bool, int64, error) {
	if info, err := os.Stat(localPath); err == nil && info.Size() > 0 {
		return false, info.Size(), nil
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		written, err := downloadBookFile(ctx, client, book.TextURL, localPath)
		if err == nil {
			return true, written, nil
		}

		lastErr = err
		if attempt == 3 {
			break
		}

		select {
		case <-ctx.Done():
			return false, 0, ctx.Err()
		case <-time.After(time.Duration(attempt) * 2 * time.Second):
		}
	}

	return false, 0, lastErr
}

func downloadBookFile(ctx context.Context, client *http.Client, textURL, localPath string) (written int64, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, textURL, nil)
	if err != nil {
		return 0, fmt.Errorf("build download request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("download text: %w", err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", closeErr)
		}
	}()

	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected download status %s", response.Status)
	}

	tmpPath := localPath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return 0, fmt.Errorf("create temp file: %w", err)
	}

	written, copyErr := io.Copy(file, response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("write file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("close file: %w", closeErr)
	}

	if err := os.Rename(tmpPath, localPath); err != nil {
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("rename file: %w", err)
	}

	return written, nil
}

func writef(writer io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(writer, format, args...)
	return err
}

func loadBookContent(path string) (int, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, "", err
	}

	text := normalizeText(string(raw))
	return len(text), text, nil
}

func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	startMarkers := []string{
		"*** START OF THE PROJECT GUTENBERG EBOOK",
		"*** START OF THIS PROJECT GUTENBERG EBOOK",
		"***START OF THE PROJECT GUTENBERG EBOOK",
	}
	endMarkers := []string{
		"*** END OF THE PROJECT GUTENBERG EBOOK",
		"*** END OF THIS PROJECT GUTENBERG EBOOK",
		"***END OF THE PROJECT GUTENBERG EBOOK",
	}

	upper := strings.ToUpper(text)
	for _, marker := range startMarkers {
		idx := strings.Index(upper, marker)
		if idx >= 0 {
			nextLine := strings.Index(text[idx:], "\n")
			if nextLine >= 0 {
				text = text[idx+nextLine+1:]
				break
			}
		}
	}

	upper = strings.ToUpper(text)
	for _, marker := range endMarkers {
		idx := strings.Index(upper, marker)
		if idx >= 0 {
			text = text[:idx]
			break
		}
	}

	lines := strings.Split(text, "\n")
	compact := make([]string, 0, len(lines))
	blankLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			blankLines++
			if blankLines > 1 {
				continue
			}
		} else {
			blankLines = 0
		}
		compact = append(compact, line)
	}

	return strings.TrimSpace(strings.Join(compact, "\n"))
}

func upsertBook(ctx context.Context, queries *store.Queries, book candidateBook, content string, contentBytes int64) error {
	subjectsJSON, err := json.Marshal(book.Subjects)
	if err != nil {
		return fmt.Errorf("marshal subjects: %w", err)
	}
	bookshelvesJSON, err := json.Marshal(book.Bookshelves)
	if err != nil {
		return fmt.Errorf("marshal bookshelves: %w", err)
	}

	err = queries.UpsertBook(ctx, store.UpsertBookParams{
		GutenbergID:   int64(book.ID),
		Title:         book.Title,
		Author:        book.Author,
		Language:      book.Language,
		Subjects:      subjectsJSON,
		Bookshelves:   bookshelvesJSON,
		DownloadCount: int32(book.DownloadCount),
		SourceUrl:     book.SourceURL,
		TextUrl:       book.TextURL,
		Content:       content,
		ContentBytes:  contentBytes,
	})
	if err != nil {
		return err
	}

	return nil
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func bytesToGigabytes(value int64) float64 {
	return float64(value) / (1024 * 1024 * 1024)
}

func ParseTargetGB(input string) (int64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(input), 64)
	if err != nil {
		return 0, fmt.Errorf("parse target GB %q: %w", input, err)
	}
	if parsed <= 0 {
		return 0, errors.New("target GB must be greater than zero")
	}
	return int64(parsed * 1024 * 1024 * 1024), nil
}

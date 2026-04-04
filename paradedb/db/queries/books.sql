-- name: UpsertBook :exec
INSERT INTO books (
	gutenberg_id,
	title,
	author,
	language,
	subjects,
	bookshelves,
	download_count,
	source_url,
	text_url,
	content,
	content_bytes,
	imported_at
)
OVERRIDING SYSTEM VALUE
VALUES (
	sqlc.arg(gutenberg_id),
	sqlc.arg(title),
	sqlc.arg(author),
	sqlc.arg(language),
	sqlc.arg(subjects)::jsonb,
	sqlc.arg(bookshelves)::jsonb,
	sqlc.arg(download_count),
	sqlc.arg(source_url),
	sqlc.arg(text_url),
	sqlc.arg(content),
	sqlc.arg(content_bytes),
	NOW()
)
ON CONFLICT (gutenberg_id)
DO UPDATE SET
	title = EXCLUDED.title,
	author = EXCLUDED.author,
	language = EXCLUDED.language,
	subjects = EXCLUDED.subjects,
	bookshelves = EXCLUDED.bookshelves,
	download_count = EXCLUDED.download_count,
	source_url = EXCLUDED.source_url,
	text_url = EXCLUDED.text_url,
	content = EXCLUDED.content,
	content_bytes = EXCLUDED.content_bytes,
	imported_at = NOW();

-- name: AnalyzeBooks :exec
ANALYZE books;
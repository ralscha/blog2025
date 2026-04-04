-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_search;

CREATE TABLE IF NOT EXISTS books (
	gutenberg_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	title TEXT NOT NULL,
	author TEXT NOT NULL,
	language TEXT NOT NULL,
	subjects JSONB NOT NULL DEFAULT '[]'::jsonb,
	bookshelves JSONB NOT NULL DEFAULT '[]'::jsonb,
	download_count INTEGER NOT NULL DEFAULT 0,
	source_url TEXT NOT NULL,
	text_url TEXT NOT NULL,
	content TEXT NOT NULL,
	content_bytes BIGINT NOT NULL DEFAULT 0,
	imported_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS books_search_idx ON books
USING bm25 (
	gutenberg_id,
	title,
	author,
	language,
	subjects,
	bookshelves,
	download_count,
	content_bytes,
	content
)
WITH (key_field = 'gutenberg_id');

-- +goose Down
DROP INDEX IF EXISTS books_search_idx;
DROP TABLE IF EXISTS books;
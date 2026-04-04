-- name: SearchBooks :many
SELECT
	gutenberg_id,
	title,
	author,
	language,
	download_count,
	content_bytes,
	pdb.score(gutenberg_id)::double precision AS score,
	CASE
		WHEN sqlc.arg(include_snippet)::boolean THEN pdb.snippet(content)::text
		ELSE ''::text
	END AS snippet
FROM books
WHERE (
	CASE
		WHEN sqlc.arg(use_conjunction)::boolean THEN (
			title &&& sqlc.arg(query_text)::text
			OR author &&& sqlc.arg(query_text)::text
			OR content &&& sqlc.arg(query_text)::text
		)
		ELSE (
			title ||| sqlc.arg(query_text)::text
			OR author ||| sqlc.arg(query_text)::text
			OR content ||| sqlc.arg(query_text)::text
		)
	END
)
AND (
	sqlc.arg(author_pattern)::text = ''
	OR author ILIKE sqlc.arg(author_pattern)::text
)
AND (
	sqlc.arg(language_filter)::text = ''
	OR language = sqlc.arg(language_filter)::text
)
ORDER BY score DESC, download_count DESC
LIMIT sqlc.arg(limit_rows)::int;
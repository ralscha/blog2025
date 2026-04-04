-- name: DemoMatchDisjunction :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	author,
	pdb.score(gutenberg_id)::text AS score
FROM books
WHERE content ||| 'love marriage'
ORDER BY score DESC
LIMIT sqlc.arg(limit_rows)::int;

-- name: DemoMatchConjunction :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	author,
	pdb.score(gutenberg_id)::text AS score
FROM books
WHERE content &&& 'love marriage'
ORDER BY score DESC
LIMIT sqlc.arg(limit_rows)::int;

-- name: DemoPhrase :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	author,
	pdb.score(gutenberg_id)::text AS score
FROM books
WHERE content ### 'single man'
ORDER BY score DESC
LIMIT sqlc.arg(limit_rows)::int;

-- name: DemoPhraseWithSlop :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	author,
	pdb.score(gutenberg_id)::text AS score
FROM books
WHERE content ### 'marriage love'::pdb.slop(2)
ORDER BY score DESC
LIMIT sqlc.arg(limit_rows)::int;

-- name: DemoTermLanguage :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	author,
	language
FROM books
WHERE language === 'en'
LIMIT sqlc.arg(limit_rows)::int;

-- name: DemoFuzzy :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	author,
	pdb.score(gutenberg_id)::text AS score
FROM books
WHERE content ||| 'marrige'::pdb.fuzzy(2)
ORDER BY score DESC
LIMIT sqlc.arg(limit_rows)::int;

-- name: DemoHighlighting :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	pdb.snippet(content)::text AS snippet
FROM books
WHERE content ||| 'marriage'
ORDER BY pdb.score(gutenberg_id) DESC
LIMIT sqlc.arg(limit_rows)::int;

-- name: DemoProximity :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	author,
	pdb.score(gutenberg_id)::text AS score
FROM books
WHERE content @@@ ('love' ## 5 ## 'marriage')
ORDER BY score DESC
LIMIT sqlc.arg(limit_rows)::int;

-- name: DemoFiltering :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	author,
	download_count::text AS download_count,
	pdb.score(gutenberg_id)::text AS score
FROM books
WHERE content ||| 'love'
	AND download_count > 100
ORDER BY score DESC
LIMIT sqlc.arg(limit_rows)::int;

-- name: DemoBoosting :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	author,
	pdb.score(gutenberg_id)::text AS score
FROM books
WHERE title ||| 'love marriage'::pdb.boost(3)
	OR content ||| 'love marriage'
ORDER BY score DESC
LIMIT sqlc.arg(limit_rows)::int;

-- name: DemoAggregateMetrics :one
SELECT
	pdb.agg('{"avg": {"field": "download_count"}}')::text AS avg_downloads,
	pdb.agg('{"value_count": {"field": "gutenberg_id"}}')::text AS total_books
FROM books
WHERE gutenberg_id @@@ pdb.all();

-- name: DemoFacets :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	author,
	pdb.agg('{"value_count": {"field": "gutenberg_id"}}') OVER ()::text AS total_matches
FROM books
WHERE content ||| 'love'
ORDER BY pdb.score(gutenberg_id) DESC
LIMIT 3;

-- name: DemoRegex :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	author
FROM books
WHERE author @@@ pdb.regex('bront.*')
LIMIT sqlc.arg(limit_rows)::int;

-- name: DemoBoolean :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	author,
	pdb.score(gutenberg_id)::text AS score
FROM books
WHERE content @@@ pdb.boolean(
	must     => ARRAY[pdb.parse('love')],
	must_not => ARRAY[pdb.parse('war')]
)
ORDER BY score DESC
LIMIT sqlc.arg(limit_rows)::int;

-- name: DemoRange :many
SELECT
	gutenberg_id::text AS gutenberg_id,
	title,
	author,
	download_count::text AS download_count
FROM books
WHERE gutenberg_id @@@ pdb.all()
	AND download_count BETWEEN 5000 AND 100000
ORDER BY download_count DESC
LIMIT sqlc.arg(limit_rows)::int;
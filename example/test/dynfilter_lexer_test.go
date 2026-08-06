package db

import (
	"slices"
	"testing"

	"example/db"
	"example/dbmysql"
	"example/dbsqlite"
)

// TestDynamicSQL_LexicalContext proves the dynamic-filter runtime only rewrites
// bind markers and interprets "-- :if $N" annotations that appear in SQL code —
// never inside string literals, quoted identifiers, or comments. Each case
// asserts both the final SQL and the final arguments.
//
// db.DynamicSQL uses the PostgreSQL runtime ($N output, [ ] is a subscript);
// dbsqlite/dbmysql exercise the SQLite bracket and MySQL backtick identifiers.
func TestDynamicSQL_LexicalContext(t *testing.T) {
	t.Run("SingleQuotedMarkers", func(t *testing.T) {
		// '?1' and '$2' are string constants and must survive untouched; only the
		// real $1 outside the strings is a placeholder.
		query := "SELECT note FROM t WHERE note = '?1' OR tag = '$2' OR a = $1"
		sql, args := db.DynamicSQL(query, []any{"x"})
		assertSQL(t, sql, "SELECT note FROM t WHERE note = '?1' OR tag = '$2' OR a = $1")
		if !slices.Equal(args, []any{"x"}) {
			t.Errorf("args: got %v, want [x]", args)
		}
	})

	t.Run("EscapedQuote", func(t *testing.T) {
		// The doubled '' keeps the literal open, so ?1 stays inside the string.
		query := "SELECT note FROM t WHERE note = 'it''s ?1' AND a = $1"
		sql, args := db.DynamicSQL(query, []any{"x"})
		assertSQL(t, sql, "SELECT note FROM t WHERE note = 'it''s ?1' AND a = $1")
		if !slices.Equal(args, []any{"x"}) {
			t.Errorf("args: got %v, want [x]", args)
		}
	})

	t.Run("LineComment", func(t *testing.T) {
		// A trailing "-- ..." comment (not a :if annotation) is preserved, and its
		// ?2/$3 are not counted as arguments.
		query := "SELECT a FROM t\nWHERE a = $1 -- drop ?2 and $3"
		sql, args := db.DynamicSQL(query, []any{"x"})
		assertSQL(t, sql, "SELECT a FROM t\nWHERE a = $1 -- drop ?2 and $3")
		if !slices.Equal(args, []any{"x"}) {
			t.Errorf("args: got %v, want [x]", args)
		}
	})

	t.Run("BlockComment", func(t *testing.T) {
		query := "SELECT /* $2 and ?3 */ a FROM t WHERE a = $1"
		sql, args := db.DynamicSQL(query, []any{"x"})
		assertSQL(t, sql, "SELECT /* $2 and ?3 */ a FROM t WHERE a = $1")
		if !slices.Equal(args, []any{"x"}) {
			t.Errorf("args: got %v, want [x]", args)
		}
	})

	t.Run("DoubleQuotedIdentifier", func(t *testing.T) {
		// $2 is part of the quoted column name, not a placeholder.
		query := `SELECT "weird$2col" FROM t WHERE a = $1`
		sql, args := db.DynamicSQL(query, []any{"x"})
		assertSQL(t, sql, `SELECT "weird$2col" FROM t WHERE a = $1`)
		if !slices.Equal(args, []any{"x"}) {
			t.Errorf("args: got %v, want [x]", args)
		}
	})

	t.Run("AnnotationInsideStringLiteral", func(t *testing.T) {
		// The "-- :if $1" is inside a string constant; it must NOT truncate the
		// line or make it conditional. A nil first arg would drop the line if the
		// annotation were (wrongly) honored.
		query := "SELECT a FROM t\nWHERE note = 'x -- :if $1 y' AND a = $1"
		sql, args := db.DynamicSQL(query, []any{"x"})
		assertSQL(t, sql, "SELECT a FROM t\nWHERE note = 'x -- :if $1 y' AND a = $1")
		if !slices.Equal(args, []any{"x"}) {
			t.Errorf("args: got %v, want [x]", args)
		}
	})

	t.Run("PostgresBracketIsSubscript", func(t *testing.T) {
		// PostgreSQL [ ] is an array subscript, so $1 inside it is a real bind.
		query := "SELECT tags[$1] FROM t WHERE a = $2"
		sql, args := db.DynamicSQL(query, []any{int64(3), "x"})
		assertSQL(t, sql, "SELECT tags[$1] FROM t WHERE a = $2")
		if !slices.Equal(args, []any{int64(3), "x"}) {
			t.Errorf("args: got %v, want [3 x]", args)
		}
	})

	t.Run("SQLiteBracketIdentifier", func(t *testing.T) {
		// SQLite [ ] quotes an identifier, so ?1/$2 inside brackets are literal
		// name characters; only the trailing ?1 is a bind marker (→ $1).
		query := "SELECT [a?1b], [x$2y] FROM t WHERE a = ?1"
		sql, args := dbsqlite.DynamicSQL(query, []any{"x"})
		assertSQL(t, sql, "SELECT [a?1b], [x$2y] FROM t WHERE a = $1")
		if !slices.Equal(args, []any{"x"}) {
			t.Errorf("args: got %v, want [x]", args)
		}
	})

	t.Run("MySQLBacktickIdentifier", func(t *testing.T) {
		// MySQL backtick-quoted identifier: the ? inside is a name character, not
		// a positional placeholder.
		query := "SELECT `weird?col` FROM t WHERE a = ?"
		sql, args := dbmysql.DynamicSQL(query, []any{"x"})
		assertSQL(t, sql, "SELECT `weird?col` FROM t WHERE a = ?")
		if !slices.Equal(args, []any{"x"}) {
			t.Errorf("args: got %v, want [x]", args)
		}
	})
}

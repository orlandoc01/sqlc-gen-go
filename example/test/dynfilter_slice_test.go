package db

import (
	"slices"
	"testing"

	"example/db"
	"example/dbmysql"
)

func TestDynamicSQLSlices(t *testing.T) {
	t.Run("PostgreSQLSqlcSlice/Repeated", func(t *testing.T) {
		query := "SELECT * FROM t WHERE id IN (/*SLICE:ids*/$1) OR parent_id IN (/*SLICE:ids*/$1)"
		gotQuery, gotArgs := db.DynamicSQL(query, []any{[]int64{7, 9}})

		assertSQL(t, gotQuery, "SELECT * FROM t WHERE id IN ($1,$2) OR parent_id IN ($1,$2)")
		if !slices.Equal(gotArgs, []any{int64(7), int64(9)}) {
			t.Errorf("args: got %v, want [7 9]", gotArgs)
		}
	})

	t.Run("SQLiteSqlcSlice/MissingArgument", func(t *testing.T) {
		query := "SELECT * FROM t WHERE id IN (/*SLICE:ids*/?2) -- :if $1"
		gotQuery, gotArgs := db.DynamicSQL(query, []any{true})

		assertSQL(t, gotQuery, "SELECT * FROM t WHERE id IN (NULL)")
		if len(gotArgs) != 0 {
			t.Errorf("args: got %v, want none", gotArgs)
		}

		gotQuery, gotArgs = db.CompileDynSQL(query).Build([]any{true})
		assertSQL(t, gotQuery, "SELECT * FROM t WHERE id IN (NULL)")
		if len(gotArgs) != 0 {
			t.Errorf("compiled args: got %v, want none", gotArgs)
		}
	})

	t.Run("MySQLSqlcSlice/ActiveCondition", func(t *testing.T) {
		query := "SELECT * FROM t WHERE kind = ? AND id IN (/*SLICE:ids*/?2) -- :if $2"
		gotQuery, gotArgs := dbmysql.DynamicSQL(query, []any{"admin", []int64{7, 9}})

		assertSQL(t, gotQuery, "SELECT * FROM t WHERE kind = ? AND id IN (?,?)")
		if !slices.Equal(gotArgs, []any{"admin", int64(7), int64(9)}) {
			t.Errorf("args: got %v, want [admin 7 9]", gotArgs)
		}
	})
}

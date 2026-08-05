package golang

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

func TestGenerateMySQLDynamicSliceRuntime(t *testing.T) {
	resp, err := Generate(context.Background(), &plugin.GenerateRequest{
		SqlcVersion: "v1.0.0",
		Settings:    &plugin.Settings{Engine: "mysql"},
		Catalog:     &plugin.Catalog{DefaultSchema: "test"},
		Queries: []*plugin.Query{{
			Name:     "SearchItems",
			Cmd:      ":many",
			Filename: "query.sql",
			Text:     "SELECT id FROM items WHERE kind = ? AND id IN (/*SLICE:ids*/?) -- :if @ids",
			Params: []*plugin.Parameter{
				{Number: 1, Column: &plugin.Column{Name: "kind", NotNull: true, Type: &plugin.Identifier{Name: "varchar"}}},
				{Number: 2, Column: &plugin.Column{Name: "ids", IsSqlcSlice: true, NotNull: true, Type: &plugin.Identifier{Name: "bigint"}}},
			},
			Columns: []*plugin.Column{{Name: "id", NotNull: true, Type: &plugin.Identifier{Name: "bigint"}}},
		}},
		PluginOptions: []byte(`{"package":"testpkg","sql_package":"database/sql","sql_driver":"github.com/go-sql-driver/mysql","emit_dynamic_filter":true}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var dynfilter string
	for _, file := range resp.Files {
		if file.Name == "dynfilter.go" {
			dynfilter = string(file.Contents)
			break
		}
	}
	if dynfilter == "" {
		t.Fatal("dynfilter.go not generated")
	}

	dir := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":       "module testpkg\n\ngo 1.23\n",
		"dynfilter.go": dynfilter,
		"dynfilter_test.go": `package testpkg

import (
	"reflect"
	"testing"
)

func TestDynamicSQL(t *testing.T) {
	query := "SELECT * FROM t WHERE kind = ? AND id IN (/*SLICE:ids*/?2) -- :if $2"
	sql, args := DynamicSQL(query, []any{"admin", []int64{7, 9}})
	if sql != "SELECT * FROM t WHERE kind = ? AND id IN (?,?)" {
		t.Fatalf("SQL: got %q", sql)
	}
	if !reflect.DeepEqual(args, []any{"admin", int64(7), int64(9)}) {
		t.Fatalf("args: got %v", args)
	}
}
`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command("go", "test", ".")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated MySQL runtime test failed: %v\n%s", err, output)
	}
	if !strings.Contains(dynfilter, "const dynQuestionMarkPlaceholders = true") {
		t.Error("generated MySQL runtime did not enable question-mark placeholders")
	}
}

package mysql

import "strings"
import "testing"

func TestPagePathSQLMatchesFilesOnly(t *testing.T) {
	if !strings.Contains(pagePathSQL, `(^|/)\.`) || !strings.Contains(pagePathSQL, `\.(css`) {
		t.Fatalf("regex escaped wrong: %s", pagePathSQL)
	}
}

package file

import (
	"testing"
)

func TestDeleteTool(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{
		"todelete.txt": "content to delete",
	})
	tool := deleteTool(fs)

	out, err := tool.Execute(t.Context(), `{"filePath": "todelete.txt"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "deleted") {
		t.Errorf("expected output to contain 'deleted', got:\n%s", out)
	}

	_, err = fs.Read("todelete.txt")
	if err == nil {
		t.Fatal("expected error reading deleted file")
	}
}

func TestDeleteToolMissingPath(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{})
	tool := deleteTool(fs)

	_, err := tool.Execute(t.Context(), `{}`)
	if err == nil {
		t.Fatal("expected error for missing filePath")
	}
}

func TestDeleteToolNotFound(t *testing.T) {
	fs := setupTestVFS(t, map[string]string{})
	tool := deleteTool(fs)

	_, err := tool.Execute(t.Context(), `{"filePath": "nonexistent.txt"}`)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

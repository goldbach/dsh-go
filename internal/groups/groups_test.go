package groups_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goldbach/dsh-go/internal/groups"
)

func writeGroup(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", filepath.Dir(filepath.Dir(dir))) // not used by Load directly

	writeGroup(t, dir, "web", `
# web servers
web1.example.com
web2.example.com
web3.example.com  # primary
`)
	writeGroup(t, dir, "db", `
db1.example.com
db2.example.com
`)

	tests := []struct {
		name  string
		files []string
		want  []string
	}{
		{
			name:  "single group",
			files: []string{filepath.Join(dir, "web")},
			want:  []string{"web1.example.com", "web2.example.com", "web3.example.com"},
		},
		{
			name:  "multiple groups",
			files: []string{filepath.Join(dir, "web"), filepath.Join(dir, "db")},
			want:  []string{"web1.example.com", "web2.example.com", "web3.example.com", "db1.example.com", "db2.example.com"},
		},
		{
			name:  "empty list",
			files: []string{},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := groups.LoadFiles(tt.files)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLoad_dedup(t *testing.T) {
	dir := t.TempDir()
	writeGroup(t, dir, "a", "host1\nhost2\n")
	writeGroup(t, dir, "b", "host2\nhost3\n")

	got, err := groups.LoadFiles([]string{filepath.Join(dir, "a"), filepath.Join(dir, "b")})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"host1", "host2", "host3"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoad_skipsBlankAndComments(t *testing.T) {
	dir := t.TempDir()
	writeGroup(t, dir, "hosts", `
# full-line comment
host1

   # indented comment
host2
host3 # inline comment
  host4
`)

	got, err := groups.LoadFiles([]string{filepath.Join(dir, "hosts")})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"host1", "host2", "host3", "host4"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoad_missingFile(t *testing.T) {
	_, err := groups.LoadFiles([]string{"/nonexistent/path/hosts"})
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_userAtHost(t *testing.T) {
	dir := t.TempDir()
	writeGroup(t, dir, "hosts", "alice@web1\nbob@web2\n")

	got, err := groups.LoadFiles([]string{filepath.Join(dir, "hosts")})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alice@web1", "bob@web2"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, got[i], want[i])
		}
	}
}

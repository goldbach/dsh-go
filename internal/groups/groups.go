package groups

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Dir returns ~/.dsh/group.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home dir: %w", err)
	}
	return filepath.Join(home, ".dsh", "group"), nil
}

// MachinesListPath returns ~/.dsh/machines.list.
func MachinesListPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home dir: %w", err)
	}
	return filepath.Join(home, ".dsh", "machines.list"), nil
}

// Load reads host names from one or more named group files under ~/.dsh/group/.
// Duplicates are removed, preserving first-seen order.
func Load(names []string) ([]string, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(names))
	for i, n := range names {
		paths[i] = filepath.Join(dir, n)
	}
	return loadFiles(paths)
}

// LoadFiles reads host names from arbitrary file paths.
func LoadFiles(paths []string) ([]string, error) {
	return loadFiles(paths)
}

func loadFiles(paths []string) ([]string, error) {
	seen := make(map[string]bool)
	var hosts []string

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open %q: %w", path, err)
		}

		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if idx := strings.IndexByte(line, '#'); idx >= 0 {
				line = strings.TrimSpace(line[:idx])
			}
			if line == "" || seen[line] {
				continue
			}
			seen[line] = true
			hosts = append(hosts, line)
		}
		f.Close()

		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("read %q: %w", path, err)
		}
	}

	return hosts, nil
}

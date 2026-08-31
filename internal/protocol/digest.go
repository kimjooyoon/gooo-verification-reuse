package protocol

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func DigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestFile(path string) (string, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	return DigestBytes(data), data, nil
}

func DigestNamedFiles(root string, names []string) (string, error) {
	type item struct {
		name string
		data []byte
	}
	items := make([]item, 0, len(names))
	for _, name := range names {
		path := filepath.Join(root, name)
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		items = append(items, item{name: filepath.ToSlash(name), data: data})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	h := sha256.New()
	for _, item := range items {
		_, _ = h.Write([]byte(item.name))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(item.data)
		_, _ = h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func TreeDigest(root string) (string, Inventory, error) {
	type entry struct {
		path string
		data []byte
	}
	entries := make([]entry, 0)
	var inventory Inventory
	err := filepath.WalkDir(root, func(path string, dirent fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if dirent.IsDir() {
			if dirent.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !dirent.Type().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == "README.md" {
			inventory.RootReadmeExcluded = true
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		entries = append(entries, entry{path: relative, data: data})
		inventory.TreeFileCount++
		inventory.TreeBytes += int64(len(data))
		switch {
		case strings.HasSuffix(relative, ".go"):
			inventory.GoFiles++
			inventory.GoLines += int64(countLines(data))
		case strings.HasSuffix(relative, ".gooo"):
			inventory.GoooFiles++
			inventory.GoooLines += int64(countLines(data))
		}
		return nil
	})
	if err != nil {
		return "", Inventory{}, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	h := sha256.New()
	for _, entry := range entries {
		_, _ = h.Write([]byte(entry.path))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(entry.data)
		_, _ = h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), inventory, nil
}

func countLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	lines := 1
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if data[len(data)-1] == '\n' {
		lines--
	}
	return lines
}

func MustDigest(value string) error {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return fmt.Errorf("invalid digest %q", value)
	}
	for _, char := range value[len("sha256:"):] {
		if !strings.ContainsRune("0123456789abcdef", char) {
			return fmt.Errorf("invalid digest %q", value)
		}
	}
	return nil
}

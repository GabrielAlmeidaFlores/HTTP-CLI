package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type fpEntry struct {
	name  string
	isDir bool
}

type filePicker struct {
	currentDir string
	entries    []fpEntry
	filtered   []fpEntry
	cursor     int
	scrollOff  int
	search     string
	filterExt  string
	onSelect   func(string)
}

func newFilePicker(onSelect func(string)) filePicker {
	fp := filePicker{onSelect: onSelect}
	startDir, err := os.UserHomeDir()
	if err != nil {
		startDir, err = os.Getwd()
		if err != nil {
			startDir = "/"
		}
	}
	fp.navigate(startDir)
	return fp
}

func (f *filePicker) navigate(dir string) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return
	}
	rawEntries, err := os.ReadDir(abs)
	if err != nil {
		return
	}
	f.currentDir = abs
	f.search = ""

	var dirs, files []fpEntry
	for _, e := range rawEntries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.IsDir() {
			dirs = append(dirs, fpEntry{name: e.Name(), isDir: true})
		} else {
			files = append(files, fpEntry{name: e.Name(), isDir: false})
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })

	f.entries = append([]fpEntry{{name: "..", isDir: true}}, append(dirs, files...)...)
	f.filtered = f.entries
	f.cursor = 0
	f.scrollOff = 0
}

func (f *filePicker) applySearch(q string) {
	f.search = q
	if q == "" {
		f.filtered = f.entries
		f.cursor = 0
		f.scrollOff = 0
		return
	}
	lower := strings.ToLower(q)
	result := []fpEntry{{name: "..", isDir: true}}
	for _, e := range f.entries[1:] {
		if strings.Contains(strings.ToLower(e.name), lower) {
			result = append(result, e)
		}
	}
	f.filtered = result
	f.cursor = 0
	f.scrollOff = 0
}

func (f *filePicker) goUp() {
	parent := filepath.Dir(f.currentDir)
	if parent == f.currentDir {
		return
	}
	prevDir := filepath.Base(f.currentDir)
	f.navigate(parent)
	for i, e := range f.filtered {
		if e.name == prevDir {
			f.cursor = i
			break
		}
	}
}

func (f *filePicker) enterSelected() bool {
	if len(f.filtered) == 0 || f.cursor >= len(f.filtered) {
		return false
	}
	e := f.filtered[f.cursor]
	if e.name == ".." {
		f.goUp()
		return true
	}
	if e.isDir {
		f.navigate(filepath.Join(f.currentDir, e.name))
		return true
	}
	return false
}

func (f *filePicker) selectedPath() string {
	if len(f.filtered) == 0 {
		return f.currentDir
	}
	e := f.filtered[f.cursor]
	if e.name == ".." {
		return filepath.Dir(f.currentDir)
	}
	return filepath.Join(f.currentDir, e.name)
}

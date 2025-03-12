// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package root

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRoot_CreateTemp(t *testing.T) {
	t.Parallel()
	sut, err := OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sut.Close()

	t.Run("nonexistent", func(t *testing.T) {
		nonexistentDir := filepath.Join("", "_not_exists_")
		f, err := sut.CreateTemp(nonexistentDir, "foo")
		if f != nil || err == nil {
			t.Errorf("CreateTemp(%q, `foo`) = %v, %v", nonexistentDir, f, err)
		}
	})

	t.Run("escapes parent", func(t *testing.T) {
		escapesDir := filepath.Join("/tmp", "_not_exists_")
		f, err := sut.CreateTemp(escapesDir, "foo")
		if f != nil || err == nil {
			t.Errorf("CreateTemp(%q, `foo`) = %v, %v", escapesDir, f, err)
			_ = os.Remove(f.Name()) // Need to os.Remove as if it escapes the parent, it's not in t.TempDir
		}
	})
}

func TestRoot_CreateTemp_Pattern(t *testing.T) {
	t.Parallel()

	tests := []struct{ pattern, prefix, suffix string }{
		{"tempfile_test", "tempfile_test", ""},
		{"tempfile_test*", "tempfile_test", ""},
		{"tempfile_test*xyz", "tempfile_test", "xyz"},
	}
	for _, test := range tests {
		sut, err := OpenRoot(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		defer sut.Close()

		f, err := sut.CreateTemp("", test.pattern)
		if err != nil {
			t.Errorf("CreateTemp(..., %q) error: %v", test.pattern, err)
			continue
		}
		base := filepath.Base(f.Name())
		f.Close()
		if !(strings.HasPrefix(base, test.prefix) && strings.HasSuffix(base, test.suffix)) {
			t.Errorf("CreateTemp pattern %q created bad name %q; want prefix %q & suffix %q",
				test.pattern, base, test.prefix, test.suffix)
		}
	}
}

func TestRoot_CreateTemp_BadPattern(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	const sep = string(os.PathSeparator)
	tests := []struct {
		pattern string
		wantErr bool
	}{
		{"ioutil*test", false},
		{"tempfile_test*foo", false},
		{"tempfile_test" + sep + "foo", true},
		{"tempfile_test*" + sep + "foo", true},
		{"tempfile_test" + sep + "*foo", true},
		{sep + "tempfile_test" + sep + "*foo", true},
		{"tempfile_test*foo" + sep, true},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			sut, err := OpenRoot(tmpDir)
			if err != nil {
				t.Fatal(err)
			}
			defer sut.Close()

			tmpfile, err := sut.CreateTemp("", tt.pattern)
			if tmpfile != nil {
				tmpfile.Close()
			}
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateTemp(..., %#q) succeeded, expected error", tt.pattern)
				}
				if !errors.Is(err, errPatternHasSeparator) {
					t.Errorf("CreateTemp(..., %#q): %v, expected ErrPatternHasSeparator", tt.pattern, err)
				}
			} else if err != nil {
				t.Errorf("CreateTemp(..., %#q): %v", tt.pattern, err)
			}
		})
	}
}

func TestRoot_MkdirTemp(t *testing.T) {
	t.Parallel()

	sut, err := OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sut.Close()

	t.Run("nonexistent", func(t *testing.T) {
		name, err := sut.MkdirTemp("_not_exists_", "foo")
		if name != "" || err == nil {
			t.Errorf("MkdirTemp(`_not_exists_`, `foo`) = %v, %v", name, err)
		}
	})

	t.Run("escapes root", func(t *testing.T) {
		name, err := sut.MkdirTemp("/tmp", "foo")
		if name != "" || err == nil {
			t.Errorf("MkdirTemp(`_not_exists_`, `foo`) = %v, %v", name, err)
			_ = os.Remove(name) // if it escapes t.TempDir then it's not going to get cleaned up automatically
		}
	})

	tests := []struct {
		pattern                string
		wantPrefix, wantSuffix string
	}{
		{"tempfile_test", "tempfile_test", ""},
		{"tempfile_test*", "tempfile_test", ""},
		{"tempfile_test*xyz", "tempfile_test", "xyz"},
		{"*xyz", "", "xyz"},
	}

	dir := "./"

	runTestMkdirTemp := func(t *testing.T, pattern, wantRePat string) {
		name, err := sut.MkdirTemp(dir, pattern)
		if name == "" || err != nil {
			t.Fatalf("MkdirTemp(dir, `tempfile_test`) = %v, %v", name, err)
		}

		re := regexp.MustCompile(wantRePat)
		if !re.MatchString(name) {
			t.Errorf("MkdirTemp(%q, %q) created bad name\n\t%q\ndid not match pattern\n\t%q", dir, pattern, name, wantRePat)
		}
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			wantRePat := "^" + regexp.QuoteMeta(dir) + regexp.QuoteMeta(tt.wantPrefix) + "[0-9]+" + regexp.QuoteMeta(tt.wantSuffix) + "$"
			runTestMkdirTemp(t, tt.pattern, wantRePat)
		})
	}
}

// test that we return a nice error message if the dir argument to TempDir doesn't
// exist (or that it's empty and TempDir doesn't exist)
func TestRoot_MkdirTemp_BadDir(t *testing.T) {
	t.Parallel()

	sut, err := OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sut.Close()

	badDir := filepath.Join(".", "not-exist")
	_, err = sut.MkdirTemp(badDir, "foo")
	if pe, ok := err.(*fs.PathError); !ok || !os.IsNotExist(err) || pe.Path != badDir {
		t.Errorf("TempDir error = %#v; want PathError for path %q satisfying IsNotExist", err, badDir)
	}
}

func TestRoot_MkdirTemp_BadPattern(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	const sep = string(os.PathSeparator)
	tests := []struct {
		pattern string
		wantErr bool
	}{
		{"ioutil*test", false},
		{"tempfile_test*foo", false},
		{"tempfile_test" + sep + "foo", true},
		{"tempfile_test*" + sep + "foo", true},
		{"tempfile_test" + sep + "*foo", true},
		{sep + "tempfile_test" + sep + "*foo", true},
		{"tempfile_test*foo" + sep, true},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			sut, err := OpenRoot(tmpDir)
			if err != nil {
				t.Fatal(err)
			}
			defer sut.Close()

			_, err = sut.MkdirTemp("", tt.pattern)
			if tt.wantErr {
				if err == nil {
					t.Errorf("MkdirTemp(..., %#q) succeeded, expected error", tt.pattern)
				}
				if !errors.Is(err, errPatternHasSeparator) {
					t.Errorf("MkdirTemp(..., %#q): %v, expected ErrPatternHasSeparator", tt.pattern, err)
				}
			} else if err != nil {
				t.Errorf("MkdirTemp(..., %#q): %v", tt.pattern, err)
			}
		})
	}
}

func TestRoot_Rename(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	tests := []struct {
		pattern string
		wantErr bool
	}{
		{"success", false},
		{"../path-escapes", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			sut, err := OpenRoot(tmpDir)
			if err != nil {
				t.Fatal(err)
			}
			defer sut.Close()

			tmpFile, err := sut.CreateTemp("", "*.tmp")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			tmpFile.Close()

			err = sut.Rename(filepath.Base(tmpFile.Name()), tt.pattern)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Rename(..., %#q) succeeded, expected error", tt.pattern)
				}
			} else if err != nil {
				t.Errorf("%s: %v", tt.pattern, err)
			}
		})
	}

	t.Run("source-escapes", func(t *testing.T) {
		sut, err := OpenRoot(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
		defer sut.Close()

		err = sut.Rename("/source-escapes", "doesnt-matter")
		if err == nil {
			t.Errorf("Rename(..., %#q) succeeded, expected error", "source-escapes")
		}
	})

	t.Run("not-found", func(t *testing.T) {
		sut, err := OpenRoot(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
		defer sut.Close()

		err = sut.Rename("not-found", "doesnt-matter")
		if err == nil {
			t.Errorf("Rename(..., %#q) succeeded, expected error", "source-escapes")
		}
	})
}

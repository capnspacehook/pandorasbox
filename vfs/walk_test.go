// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vfs

import (
	"io/fs"
	"os"
	pathpkg "path"
	"testing"
)

type Node struct {
	name    string
	entries []*Node // nil if the entry is a file
	mark    int
}

var tree = &Node{
	"testdata",
	[]*Node{
		{"a", nil, 0},
		{"b", []*Node{}, 0},
		{"c", nil, 0},
		{
			"d",
			[]*Node{
				{"x", nil, 0},
				{"y", []*Node{}, 0},
				{
					"z",
					[]*Node{
						{"u", nil, 0},
						{"v", nil, 0},
					},
					0,
				},
			},
			0,
		},
	},
	0,
}

func walkTree(n *Node, path string, f func(path string, n *Node)) {
	f(path, n)
	for _, e := range n.entries {
		walkTree(e, pathpkg.Join(path, e.name), f)
	}
}

func makeTree(t *testing.T) fs.FS {
	t.Helper()

	fsys := NewFS()
	walkTree(tree, tree.name, func(path string, n *Node) {
		if n.entries == nil {
			f, err := fsys.Create(path)
			if err != nil {
				t.Fatalf("error creating file %s: %v", path, err)
			}
			if err := f.Close(); err != nil {
				t.Fatalf("error closing file %s: %v", path, err)
			}
		} else {
			if err := fsys.Mkdir(path, 0o755); err != nil {
				t.Fatalf("error creating dir %s: %v", path, err)
			}
		}
	})
	return fsys.FS()
}

func checkMarks(t *testing.T, report bool) {
	t.Helper()

	walkTree(tree, tree.name, func(path string, n *Node) {
		if n.mark != 1 && report {
			t.Errorf("node %s mark = %d; expected 1", path, n.mark)
		}
		n.mark = 0
	})
}

// Assumes that each node name is unique. Good enough for a test.
// If err if not nil, it is appended to errors.
func mark(entry fs.DirEntry, err error, errors *[]error) error {
	name := entry.Name()
	walkTree(tree, tree.name, func(path string, n *Node) {
		if n.name == name {
			n.mark++
		}
	})
	if err != nil {
		*errors = append(*errors, err)
		return nil
	}
	return nil
}

func TestWalkDir(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal("finding working dir:", err)
	}
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatal("entering temp dir:", err)
	}
	defer func() {
		err := os.Chdir(origDir)
		if err != nil {
			t.Error(err)
		}
	}()

	fsys := makeTree(t)
	errors := make([]error, 0, 10)
	markFn := func(path string, entry fs.DirEntry, err error) error {
		return mark(entry, err, &errors)
	}
	// Expect no errors.
	err = fs.WalkDir(fsys, ".", markFn)
	if err != nil {
		t.Fatalf("no error expected, found: %s", err)
	}
	if len(errors) != 0 {
		t.Fatalf("unexpected errors: %s", errors)
	}
	checkMarks(t, true)
}

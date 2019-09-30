// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ioutil

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/capnspacehook/pandorasbox/osfs"
)

func ExampleReadAll() {
	r := strings.NewReader("Go is a general-purpose language designed with systems programming in mind.")

	b, err := ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s", b)

	// Output:
	// Go is a general-purpose language designed with systems programming in mind.
}

func ExampleReadDir() {
	fs, _ := osfs.NewFS()

	files, err := ReadDir(fs, ".")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		fmt.Println(file.Name())
	}
}

func ExampleTempDir() {
	fs, _ := osfs.NewFS()
	content := []byte("temporary file's content")
	dir, err := TempDir(fs, "", "example")
	if err != nil {
		log.Fatal(err)
	}

	defer fs.RemoveAll(dir) // clean up

	tmpfn := filepath.Join(dir, "tmpfile")
	if err := WriteFile(fs, tmpfn, content, 0666); err != nil {
		log.Fatal(err)
	}
}

func ExampleTempFile() {
	fs, _ := osfs.NewFS()
	content := []byte("temporary file's content")
	tmpfile, err := TempFile(fs, "", "example")
	if err != nil {
		log.Fatal(err)
	}

	defer fs.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(content); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}
}

func ExampleReadFile() {
	fs, _ := osfs.NewFS()

	content, err := ReadFile(fs, "testdata/hello")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("File contents: %s", content)

	// Output:
	// File contents: Hello, Gophers!
}

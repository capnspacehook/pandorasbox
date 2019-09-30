package main

import (
	"fmt"

	"github.com/capnspacehook/pandorasbox"
)

func main() {
	box, err := pandorasbox.NewBox()
	if err != nil {
		panic(err)
	}

	if err = box.WriteFile("vfs://file.txt", []byte("Testing testing 1 2 3"), 0644); err != nil {
		panic(err)
	}

	data, err := box.ReadFile("vfs://file.txt")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))
}

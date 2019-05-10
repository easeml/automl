package dataset

import (
	"archive/tar"
	"bytes"
	"io"
	"log"
	"testing"
)

func TestTarOpener(t *testing.T) {

	// Construct file structure.
	var err error
	var rw io.ReadWriteCloser
	outputOpener := NewTarOpener()
	_, err = outputOpener.GetDir("root", "dir1", false)
	if err != nil {
		log.Fatal(err)
	}
	_, err = outputOpener.GetDir("root", "dir2", false)
	if err != nil {
		log.Fatal(err)
	}
	rw, err = outputOpener.GetFile("root", "dir1/file1.txt", false, false)
	if err != nil {
		log.Fatal(err)
	}
	file1 := []byte("File 1 Contents")
	_, err = rw.Write(file1)
	if err != nil {
		log.Fatal(err)
	}
	rw, err = outputOpener.GetFile("root", "dir1/file2.txt", false, false)
	if err != nil {
		log.Fatal(err)
	}
	file2 := []byte("File 2 Contents")
	_, err = rw.Write(file2)
	if err != nil {
		log.Fatal(err)
	}

	// Archive into tar.
	reader, err := DumpTarOpener(outputOpener)
	if err != nil {
		log.Fatal(err)
	}

	// Extract from tar.
	tarReader := tar.NewReader(reader)
	inputOpener, err := LoadTarOpener(tarReader)
	if err != nil {
		log.Fatal(err)
	}
	children, err := inputOpener.GetDir("", "root", true)
	if err != nil {
		log.Fatal(err)
	}
	if len(children) != 2 || (children[0] != "dir1" && children[1] != "dir1") || (children[0] != "dir2" && children[1] != "dir2") {
		log.Fatalf("Expected children [\"dir1\", \"dir2\"], found: %s", children)
	}
	children, err = inputOpener.GetDir("root", "dir1", true)
	if err != nil {
		log.Fatal(err)
	}
	if len(children) != 2 || (children[0] != "file1.txt" && children[1] != "file1.txt") || (children[0] != "file2.txt" && children[1] != "file2.txt") {
		log.Fatalf("Expected children [\"file1.txt\", \"file2.txt\"], found: %s", children)
	}
	children, err = inputOpener.GetDir("root", "dir2", true)
	if err != nil {
		log.Fatal(err)
	}
	if len(children) != 0 {
		log.Fatalf("Expected 0 children, found: %s", children)
	}
	rw, err = outputOpener.GetFile("root", "dir1/file1.txt", true, false)
	if err != nil {
		log.Fatal(err)
	}
	var file1Input bytes.Buffer
	_, err = io.Copy(&file1Input, rw)
	if err != nil {
		log.Fatal(err)
	}
	file1B := file1Input.Bytes()
	if string(file1B) != string(file1) {
		log.Fatalf("Expected file content to be \"%s\", found: \"%s\"", string(file1), string(file1B))
	}
	rw, err = outputOpener.GetFile("root", "dir1/file2.txt", true, false)
	if err != nil {
		log.Fatal(err)
	}
	var file2Input bytes.Buffer
	_, err = io.Copy(&file2Input, rw)
	if err != nil {
		log.Fatal(err)
	}
	file2B := file2Input.Bytes()
	if string(file2B) != string(file2) {
		log.Fatalf("Expected file content to be \"%s\", found: \"%s\"", string(file2), string(file2B))
	}

}

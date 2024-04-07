package algo_test

import (
	"io"
	"os"
	"testing"

	"github.com/a13labs/cobot/internal/algo"
)

func TestBinaryFileStream(t *testing.T) {
	// Create a temporary file for testing
	f, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	// Create a new BinaryFileStream
	bfs, err := algo.NewBinaryFileStream(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer bfs.Close()

	// Test WriteString and ReadString
	testString := "Hello, World!"
	err = bfs.WriteString(testString)
	if err != nil {
		t.Fatal(err)
	}

	_, err = bfs.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}

	readString, err := bfs.ReadString(len(testString))
	if err != nil {
		t.Fatal(err)
	}
	if readString != testString {
		t.Errorf("ReadString() = %v; want %v", readString, testString)
	}

	// Test WriteFloat64 and ReadFloat64
	testFloat := 3.14159
	err = bfs.WriteFloat64(testFloat)
	if err != nil {
		t.Fatal(err)
	}

	_, err = bfs.Seek(-8, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}

	readFloat, err := bfs.ReadFloat64()
	if err != nil {
		t.Fatal(err)
	}
	if readFloat != testFloat {
		t.Errorf("ReadFloat64() = %v; want %v", readFloat, testFloat)
	}
}

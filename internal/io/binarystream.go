package io

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"os"
)

type BinaryFileStream struct {
	file *os.File
}

func NewBinaryFileStream(path string) (*BinaryFileStream, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	return &BinaryFileStream{file: f}, nil
}

func (b *BinaryFileStream) WriteInt32(i int32) error {
	return binary.Write(b.file, binary.LittleEndian, i)
}

func (b *BinaryFileStream) ReadInt32() (int32, error) {
	var i int32
	err := binary.Read(b.file, binary.LittleEndian, &i)
	return i, err
}

func (b *BinaryFileStream) WriteInt64(i int64) error {
	return binary.Write(b.file, binary.LittleEndian, i)
}

func (b *BinaryFileStream) ReadInt64() (int64, error) {
	var i int64
	err := binary.Read(b.file, binary.LittleEndian, &i)
	return i, err
}

func (b *BinaryFileStream) WriteFloat64(f float64) error {
	return binary.Write(b.file, binary.LittleEndian, f)
}

func (b *BinaryFileStream) ReadFloat64() (float64, error) {
	var f float64
	err := binary.Read(b.file, binary.LittleEndian, &f)
	return f, err
}

func (b *BinaryFileStream) WriteString(s string) (int, error) {
	return b.file.WriteString(s)
}

func (b *BinaryFileStream) ReadString(n int) (string, error) {
	buf := make([]byte, n)
	_, err := b.file.Read(buf)
	return string(buf), err
}

func (b *BinaryFileStream) Write(p []byte) (int, error) {
	return b.file.Write(p)
}

func (b *BinaryFileStream) Read(p []byte) (int, error) {
	return b.file.Read(p)
}

func (b *BinaryFileStream) ReadAt(p []byte, off int64) (int, error) {
	return b.file.ReadAt(p, off)
}

func (b *BinaryFileStream) WriteAt(p []byte, off int64) (int, error) {
	return b.file.WriteAt(p, off)
}

func (b *BinaryFileStream) Truncate(size int64) error {
	return b.file.Truncate(size)
}

func (b *BinaryFileStream) SeekCurrent() (int64, error) {
	return b.file.Seek(0, 1)
}

func (b *BinaryFileStream) WriteData(data any) (int, error) {
	// Encode the struct into a byte slice
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return 0, err
	}

	// Convert the buffer to a byte slice
	return b.file.Write(buf.Bytes())
}

func (b *BinaryFileStream) ReadData(data any) error {

	// length := unsafe.Sizeof(*data) + 4
	// buf := make([]byte, length)

	// // Read the data from the file
	// if _, err := b.file.Read(buf); err != nil {
	// 	return err
	// }

	// Decode the data
	// inBuf := bytes.NewBuffer(buf)

	dec := gob.NewDecoder(b.file)

	if err := dec.Decode(data); err != nil {
		return err
	}

	return nil
}

func (b *BinaryFileStream) SeekEnd() (int64, error) {
	return b.file.Seek(0, 2)
}

func (b *BinaryFileStream) SeekStart() error {
	_, err := b.file.Seek(0, 0)
	return err
}

func (b *BinaryFileStream) Seek(offset int64, whence int) (int64, error) {
	return b.file.Seek(offset, whence)
}

func (b *BinaryFileStream) Close() error {
	return b.file.Close()
}

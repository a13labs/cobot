package algo

import (
	"encoding/binary"
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

func (b *BinaryFileStream) WriteString(s string) error {
	_, err := b.file.WriteString(s)
	return err
}

func (b *BinaryFileStream) ReadString(n int) (string, error) {
	buf := make([]byte, n)
	_, err := b.file.Read(buf)
	return string(buf), err
}

func (b *BinaryFileStream) Seek(offset int64, whence int) (int64, error) {
	return b.file.Seek(offset, whence)
}

func (b *BinaryFileStream) Close() error {
	return b.file.Close()
}

package uploader

import (
	"hash/crc32"
	"io"
)

// CalculateCRC32 computes the CRC32 hash of the given reader
func CalculateCRC32(r io.Reader) (uint32, error) {
    hash := crc32.NewIEEE()
    if _, err := io.Copy(hash, r); err != nil {
        return 0, err
    }
    return hash.Sum32(), nil
}
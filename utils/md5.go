package utils

import (
	"crypto/md5"
	"encoding/hex"
	"io"
)

// CalculateMD5 computes the MD5 hash of the given reader
func CalculateMD5(r io.Reader) (string, error) {
    hash := md5.New()
    if _, err := io.Copy(hash, r); err != nil {
        return "", err
    }
    return hex.EncodeToString(hash.Sum(nil)), nil
}
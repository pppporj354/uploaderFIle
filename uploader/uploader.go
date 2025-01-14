package uploader

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"hash"
	"hash/crc32"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"
)

type FileUploader struct {
    maxRetries  int
    retryDelay  time.Duration
    client      *http.Client
    usemd5      bool
}

func NewFileUploader(maxRetries int, retryDelay time.Duration, usemd5 bool) *FileUploader {
    return &FileUploader{
        maxRetries: maxRetries,
        retryDelay: retryDelay,
        client:     &http.Client{},
        usemd5:     usemd5,
    }
}

func (u *FileUploader) Upload(ctx context.Context, filepath, url string) error {
    file, err := os.Open(filepath)
    if err != nil {
        return err
    }
    defer file.Close()

    // Create buffer for the multipart form
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    part, err := writer.CreateFormFile("file", filepath)
    if err != nil {
        return err
    }

    // Setup multiple hash calculations
    crc32Hash := crc32.NewIEEE()
    var md5Hash hash.Hash
    var tee io.Reader

    if u.usemd5 {
        md5Hash = md5.New()
        tee = io.TeeReader(file, io.MultiWriter(crc32Hash, md5Hash))
    } else {
        tee = io.TeeReader(file, crc32Hash)
    }

    // Copy file content
    _, err = io.Copy(part, tee)
    if err != nil {
        return err
    }

    writer.Close()

    // Create request with context
    req, err := http.NewRequestWithContext(ctx, "POST", url, body)
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", writer.FormDataContentType())
    req.Header.Set("CRC32", strconv.FormatUint(uint64(crc32Hash.Sum32()), 10))
    
    if u.usemd5 {
        md5str := hex.EncodeToString(md5Hash.Sum(nil))
        req.Header.Set("Content-MD5", md5str)
    }

    return u.doWithRetry(ctx, req)
}
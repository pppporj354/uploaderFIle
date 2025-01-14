package uploader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestUploadManager(t *testing.T) {
    // Create a test server with atomic request counter
    var requestCount int32
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        atomic.AddInt32(&requestCount, 1)
        
        // Verify request is multipart
        if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
            w.WriteHeader(http.StatusBadRequest)
            return
        }

        // Read file content
        file, _, err := r.FormFile("file")
        if err != nil {
            w.WriteHeader(http.StatusBadRequest)
            return
        }
        defer file.Close()

        // Simulate processing time
        time.Sleep(100 * time.Millisecond)
        
        // Discard file content
        _, err = io.Copy(io.Discard, file)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    // Create temporary test files
    numFiles := 3
    files := make([]string, numFiles)
    for i := 0; i < numFiles; i++ {
        tmpFile, err := os.CreateTemp("", fmt.Sprintf("test-file-%d-*.txt", i))
        if err != nil {
            t.Fatal(err)
        }
        defer os.Remove(tmpFile.Name())
        
        // Write test content
        content := fmt.Sprintf("test content %d", i)
        if _, err := tmpFile.WriteString(content); err != nil {
            t.Fatal(err)
        }
        if err := tmpFile.Close(); err != nil {
            t.Fatal(err)
        }
        files[i] = tmpFile.Name()
    }

    // Create upload tasks
    tasks := make([]UploadTask, len(files))
    for i, file := range files {
        tasks[i] = UploadTask{
            FilePath: file,
            URL:      server.URL,
            Result:   make(chan error, 1),
        }
    }

    // Create manager with 2 workers
    manager := NewUploadManager(2, 3, 100*time.Millisecond, false)
    defer manager.Close()

    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Measure upload time
    start := time.Now()
    errors := manager.UploadFiles(ctx, tasks)
    duration := time.Since(start)

    // Test assertions
    t.Run("All files uploaded", func(t *testing.T) {
        if int(atomic.LoadInt32(&requestCount)) != len(files) {
            t.Errorf("Expected %d requests, got %d", len(files), requestCount)
        }
    })

    t.Run("No errors occurred", func(t *testing.T) {
        for i, err := range errors {
            if err != nil {
                t.Errorf("Error uploading file %d: %v", i, err)
            }
        }
    })

 t.Run("Concurrent execution verified", func(t *testing.T) {
    // With 2 workers and 3 files, we expect roughly 200ms (2 batches of 100ms each)
    minDuration := 150 * time.Millisecond // Allow some margin
    maxDuration := 250 * time.Millisecond
    sequential := time.Duration(numFiles) * 100 * time.Millisecond

    if duration < minDuration {
        t.Errorf("Execution too fast (%v), suggests files weren't processed", duration)
    }
    if duration > maxDuration {
        t.Errorf("Execution too slow (%v), expected < %v", duration, maxDuration)
    }
    if duration >= sequential {
        t.Errorf("No concurrent benefit: %v >= %v (sequential time)", duration, sequential)
    }
})
}
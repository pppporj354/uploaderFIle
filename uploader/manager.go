package uploader

import (
	"context"
	"sync"
	"time"
)

type UploadTask struct {
    FilePath string
    URL      string
    Result   chan error
}

type UploadManager struct {
    uploader *FileUploader
    workers  int
    queue    chan UploadTask
    wg       sync.WaitGroup
}

func NewUploadManager(workers int, maxRetries int, retryDelay time.Duration, usemd5 bool) *UploadManager {
    manager := &UploadManager{
        uploader: NewFileUploader(maxRetries, retryDelay, usemd5),
        workers:  workers,
        queue:    make(chan UploadTask),
    }
    manager.start()
    return manager
}

func (m *UploadManager) start() {
    for i := 0; i < m.workers; i++ {
        m.wg.Add(1)
        go m.worker()
    }
}

func (m *UploadManager) worker() {
    defer m.wg.Done()
    for task := range m.queue {
        err := m.uploader.Upload(context.Background(), task.FilePath, task.URL)
        task.Result <- err
    }
}

func (m *UploadManager) UploadFiles(ctx context.Context, files []UploadTask) []error {
    errors := make([]error, len(files))
    var wg sync.WaitGroup
    
   
    for i := range files {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            select {
            case <-ctx.Done():
                errors[i] = ctx.Err()
            case m.queue <- files[i]:
                errors[i] = <-files[i].Result
            }
        }(i)
    }
    
    wg.Wait()
    return errors
}

func (m *UploadManager) Close() {
    close(m.queue)
    m.wg.Wait()
}
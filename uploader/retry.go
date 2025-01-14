package uploader

import (
	"context"
	"net/http"
	"time"
)

func (u *FileUploader) doWithRetry(ctx context.Context, req *http.Request) error {
    var lastErr error
    
    for attempt := 0; attempt <= u.maxRetries; attempt++ {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            resp, err := u.client.Do(req)
            if err != nil {
                lastErr = err
                time.Sleep(u.retryDelay * time.Duration(attempt+1))
                continue
            }
            defer resp.Body.Close()
            
            if resp.StatusCode >= 200 && resp.StatusCode < 300 {
                return nil
            }
            
            time.Sleep(u.retryDelay * time.Duration(attempt+1))
        }
    }
    
    return lastErr
}
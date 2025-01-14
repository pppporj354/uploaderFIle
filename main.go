// package main

// import (
// 	"context"
// 	"log"
// 	"net/http"
// 	"net/http/httptest"
// 	"time"

// 	"uploaderFile/uploader"
// )

// func main() {
//     // Create a test server
//  server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//         log.Printf("Received file upload request")
//         log.Printf("CRC32: %s", r.Header.Get("CRC32"))
//         log.Printf("MD5: %s", r.Header.Get("Content-MD5"))
//         w.WriteHeader(http.StatusOK)
//     }))
//     defer server.Close()

//     // Create upload manager with 3 workers
//     manager := uploader.NewUploadManager(3, 3, time.Second, true)
//     defer manager.Close()

//     // Create context with timeout
//     ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//     defer cancel()

//     // Prepare upload tasks
//     files := []uploader.UploadTask{
//         {
//             FilePath: "./testfile1.txt",
//             URL:      server.URL,
//             Result:   make(chan error, 1),
//         },
//         {
//             FilePath: "./testfile2.txt",
//             URL:      server.URL,
//             Result:   make(chan error, 1),
//         },
//         // Add more files as needed
//     }

//     // Upload files concurrently
//     errors := manager.UploadFiles(ctx, files)

//     // Check for errors
//     for i, err := range errors {
//         if err != nil {
//             log.Printf("Error uploading file %s: %v", files[i].FilePath, err)
//         } else {
//             log.Printf("Successfully uploaded file %s", files[i].FilePath)
//         }
//     }
// }

package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"uploaderFile/uploader"
)

type PageData struct {
    Success bool
    Message string
    Error   string
}

func main() {
    // Create upload manager with 3 workers
    manager := uploader.NewUploadManager(3, 3, time.Second, true)
    defer manager.Close()

    // Load template
    tmpl, err := template.ParseFiles("templates/base.html")
    if err != nil {
        log.Fatal(err)
    }

    // Handle index page
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        tmpl.Execute(w, &PageData{})
    })

    // Handle file upload
    http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        // Parse multipart form
        err := r.ParseMultipartForm(10 << 20) // 10 MB max
        if err != nil {
            tmpl.Execute(w, &PageData{Error: "File too large"})
            return
        }

        files := r.MultipartForm.File["file"]
        if len(files) == 0 {
            tmpl.Execute(w, &PageData{Error: "No files selected"})
            return
        }

        // Prepare upload tasks
        tasks := make([]uploader.UploadTask, len(files))
        for i, fileHeader := range files {
            // Save file temporarily
            dst := filepath.Join("uploads", fileHeader.Filename)
            file, err := fileHeader.Open()
            if err != nil {
                tmpl.Execute(w, &PageData{Error: "Error processing file"})
                return
            }
            defer file.Close()

            tasks[i] = uploader.UploadTask{
                FilePath: dst,
                URL:      "http://your-upload-url", // Replace with your upload URL
                Result:   make(chan error, 1),
            }
        }

        // Upload files
        errors := manager.UploadFiles(r.Context(), tasks)

        // Check for errors
        for _, err := range errors {
            if err != nil {
                tmpl.Execute(w, &PageData{Error: "Upload failed: " + err.Error()})
                return
            }
        }

        tmpl.Execute(w, &PageData{
            Success: true,
            Message: "Files uploaded successfully!",
        })
    })

    // Create uploads directory if it doesn't exist
    if err := os.MkdirAll("uploads", 0755); err != nil {
        log.Fatal(err)
    }

    log.Println("Server starting on http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
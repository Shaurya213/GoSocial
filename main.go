package main

import (
    "bufio"
    "context"
    "fmt"
    "io"
    "log"
    "mime"
    "os"
    "path/filepath"
    "strings"

    "gosocial/internal/config"
    "gosocial/internal/dbmongo"
)

func main() {
    // Setup
    cfg := config.LoadConfig()
    mc, err := dbmongo.NewMongoConnection(cfg)
    if err != nil {
        log.Fatal("Connection failed:", err)
    }
    defer mc.Close(context.Background())

    storage := dbmongo.NewMediaStorage(mc)
    
    // Simple input choice
    reader := bufio.NewReader(os.Stdin)
    fmt.Print("Upload or Download? (u/d): ")
    choice, _ := reader.ReadString('\n')
    choice = strings.TrimSpace(strings.ToLower(choice))

    if choice == "u" {
        // UPLOAD PATH
        fmt.Print("Enter filename: ")
        filename, _ := reader.ReadString('\n')
        filename = strings.TrimSpace(filename)
        
        file, err := os.Open(filename)
        if err != nil {
            log.Fatal("File not found:", err)
        }
        defer file.Close()

        stat, _ := file.Stat()
        contentType := mime.TypeByExtension(filepath.Ext(filename))
        
        uploaded, err := storage.UploadFile(context.Background(), stat.Name(), contentType, "user123", file)
        if err != nil {
            log.Fatal("Upload failed:", err)
        }

        fmt.Printf("✅ Uploaded! ID: %s\n", uploaded.ID)
        
    } else if choice == "d" {
        // DOWNLOAD PATH
        fmt.Print("Enter file ID: ")
        fileID, _ := reader.ReadString('\n')
        fileID = strings.TrimSpace(fileID)
        
        reader_stream, fileInfo, err := storage.DownloadFile(context.Background(), fileID)
        if err != nil {
            log.Fatal("Download failed:", err)
        }

        // Save with ID as filename
        outputFile := fmt.Sprintf("%s%s", fileID, filepath.Ext(fileInfo.Filename))
        file, err := os.Create(outputFile)
        if err != nil {
            log.Fatal("Create failed:", err)
        }
        defer file.Close()

        io.Copy(file, reader_stream)
        fmt.Printf("✅ Downloaded as: %s\n", outputFile)
        
    } else {
        fmt.Println("Invalid choice")
    }
}


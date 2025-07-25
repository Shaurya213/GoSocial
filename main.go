package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"gosocial/internal/config"
	"gosocial/internal/dbmongo"
)

// simple coloured output (TUI-ish without extra libs)
const (
	green = "\033[32m"
	cyan  = "\033[36m"
	reset = "\033[0m"
)

func main() {
	// -------- 1. CONFIG & CONNECTION ---------------------------------
	cfg := config.LoadConfig()
	mc, err := dbmongo.NewMongoConnection(cfg)
	must(err)
	defer mc.Close(context.Background())

	// create storage helper with base URL (optional)
	baseURL := "http://localhost:8080/media"
	storage := dbmongo.NewMediaStorage(mc, baseURL)

	// -------- 2. PICK A REAL FILE -----------------------------------
	const localFile = "sample.jpg" // change if needed
	f, err := os.Open(localFile)
	must(err)
	defer f.Close()

	stat, _ := f.Stat()
	display(fmt.Sprintf("Uploading %s (%d bytes)…", stat.Name(), stat.Size()), cyan)

	// -------- 3. UPLOAD TO GRIDFS -----------------------------------
	uploaded, err := storage.UploadMediaWithMetadata(
		context.Background(),
		stat.Name(),
		detectMIME(stat.Name()),
		"cli-tester", // uploaderID for demo
		f,
	)
	must(err)
	display("✔ uploaded", green)

	// -------- 4. DOWNLOAD IT BACK -----------------------------------
	r, meta, err := storage.DownloadMediaWithInfo(context.Background(), uploaded.ID)
	must(err)

	buf := make([]byte, 40)               // preview first 40 bytes
	n, _ := io.ReadFull(r, buf[:])
	preview := fmt.Sprintf("% x", buf[:n]) // hex view

	// -------- 5. MINIMAL TUI OUTPUT ---------------------------------
	fmt.Println(strings.Repeat("─", 58))
	fmt.Printf("%sFile:%s   %s\n", cyan, reset, meta.Filename)
	fmt.Printf("%sSize:%s   %d bytes\n", cyan, reset, meta.Size)
	fmt.Printf("%sGridFS ID:%s %s\n", cyan, reset, meta.ID)
	fmt.Printf("%sURL:%s    %s\n", cyan, reset, meta.URL)
	fmt.Printf("%sPreview:%s %s …\n", cyan, reset, preview)
	fmt.Println(strings.Repeat("─", 58))
}

// detectMIME – very small helper; real code may use mime.TypeByExtension
func detectMIME(name string) string {
	switch {
	case strings.HasSuffix(name, ".jpg"), strings.HasSuffix(name, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	case strings.HasSuffix(name, ".mp4"):
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// tiny helper for coloured log
func display(msg, colour string) {
	fmt.Printf("%s%s%s  %s\n", colour, time.Now().Format("15:04:05"), reset, msg)
}


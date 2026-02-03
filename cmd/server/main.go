package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gnemet/SlideForge/internal/pptx"
	"github.com/gnemet/datagrid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var (
	db   *sql.DB
	tmpl *template.Template
)

func main() {
	godotenv.Load()

	// Database Connection
	var err error
	db, err = sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize Directories
	os.MkdirAll("uploads", 0755)
	os.MkdirAll("thumbnails", 0755)

	// Templates
	tmpl = template.Must(template.ParseGlob("ui/templates/*.html"))

	// Datagrid Handler for PPTX Files
	dgHandler := datagrid.NewHandler(db, "pptx_files", []datagrid.UIColumn{
		{Field: "filename", Label: "Name", Visible: true, Sortable: true},
		{Field: "created_at", Label: "Uploaded", Visible: true, Sortable: true},
		{Field: "is_template", Label: "Template", Visible: true, Type: "boolean"},
	}, datagrid.DatagridConfig{})

	// Routes
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("ui/static"))))
	http.Handle("/thumbnails/", http.StripPrefix("/thumbnails/", http.FileServer(http.Dir("thumbnails"))))

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/dashboard", handleDashboard)
	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/templates", dgHandler.ServeHTTP)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("SlideForge starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	tmpl.ExecuteTemplate(w, "base.html", nil)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "dashboard.html", nil)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	destPath := filepath.Join("uploads", header.Filename)
	dest, err := os.Create(destPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dest.Close()
	io.Copy(dest, file)

	// Process PPTX to PNGs
	thumbDir := filepath.Join("thumbnails", header.Filename)
	_, err = pptx.ExtractSlidesToPNG(destPath, thumbDir)
	if err != nil {
		log.Printf("PNG extraction failed: %v", err)
	}

	// Insert into DB
	_, err = db.Exec("INSERT INTO pptx_files (filename, original_file_path, thumbnail_dir_path) VALUES ($1, $2, $3)",
		header.Filename, destPath, thumbDir)
	if err != nil {
		log.Printf("DB insert failed: %v", err)
	}

	// Return dashboard partial
	handleDashboard(w, r)
}

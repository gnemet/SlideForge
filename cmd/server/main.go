package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"encoding/json"
	"sync"
	"time"

	"github.com/gnemet/SlideForge/internal/ai"
	"github.com/gnemet/SlideForge/internal/assets"
	"github.com/gnemet/SlideForge/internal/config"
	"github.com/gnemet/SlideForge/internal/database"
	"github.com/gnemet/SlideForge/internal/i18n"
	"github.com/gnemet/SlideForge/internal/observer"
	"github.com/gnemet/SlideForge/internal/pptx"
	"github.com/gnemet/datagrid"
	"github.com/russross/blackfriday/v2"
)

var (
	sqlDB      *sql.DB
	tmpl       *template.Template
	cfg        *config.Config
	aiClient   *ai.Client
	obs        *observer.Observer
	logChan    chan string
	logBuffer  []string
	logMutex   sync.Mutex
	sseClients = make(map[chan string]bool)
	sseMutex   sync.Mutex
)

func main() {
	var err error
	cfg, err = config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	aiClient = ai.NewClient(cfg)

	// Initialize Log Channel
	logChan = make(chan string, 100)
	go processLogs()

	// Database Connection - Jiramntr Style
	sqlDB, err = database.NewConnection(cfg.Database.GetConnectStr())
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	// Set search path explicitly - Jiramntr Style
	if cfg.Database.Options != "" {
		_, err = sqlDB.Exec(fmt.Sprintf("SET %s", strings.TrimPrefix(cfg.Database.Options, "-c ")))
		if err != nil {
			log.Printf("Warning: Failed to set search_path: %v", err)
		}
	} else {
		// Default search path for SlideForge
		_, err = sqlDB.Exec("SET search_path TO slideforge, public")
		if err != nil {
			log.Printf("Warning: Failed to set default search_path: %v", err)
		}
	}

	log.Println("Database connection established")

	// Start Background Observer
	obs = observer.NewObserver(cfg, sqlDB, aiClient, logChan)

	go func() {
		if err := obs.Start(context.Background()); err != nil {
			log.Printf("Observer error: %v", err)
		}
	}()

	// Initialize Directories
	os.MkdirAll(cfg.Application.Storage.Stage, 0755)
	os.MkdirAll(cfg.Application.Storage.Thumbnails, 0755)

	i18nFS, _ := fs.Sub(assets.EmbeddedAssets, "resources")
	i18n.Init(i18nFS)

	// Templates - Parse layout and datagrid templates
	funcMap := template.FuncMap{
		"T": func(lang, key string) string {
			return i18n.T(lang, key)
		},
		"stripExt": func(filename string) string {
			return strings.TrimSuffix(filename, filepath.Ext(filename))
		},
		"mod": func(a, b int) int {
			return a % b
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}
	// Merge Datagrid Template Funcs (renderRow, etc.)
	for k, v := range datagrid.TemplateFuncs() {
		funcMap[k] = v
	}

	// Load templates from embedded FS
	tmpl = template.Must(template.New("").Funcs(funcMap).ParseFS(assets.EmbeddedAssets, "ui/templates/layout/*.html", "ui/templates/partials/*.html"))
	// Automagically include all datagrid library templates
	tmpl = template.Must(tmpl.ParseFS(datagrid.UIAssets, "ui/templates/partials/datagrid/*.html"))

	// Datagrid Handler for PPTX Files
	dgHandler := datagrid.NewHandler(sqlDB, "pptx_files", []datagrid.UIColumn{
		{Field: "filename", Label: "Name", Visible: true, Sortable: true},
		{Field: "created_at", Label: "Uploaded", Visible: true, Sortable: true},
		{Field: "is_template", Label: "Template", Visible: true, Type: "boolean"},
	}, datagrid.DatagridConfig{})

	// Static files - Preferred from embedded FS for portability
	staticUI, _ := fs.Sub(assets.EmbeddedAssets, "ui/static")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticUI))))

	log.Printf("Serving thumbnails from: %s", cfg.Application.Storage.Thumbnails)
	http.Handle("/thumbnails/", http.StripPrefix("/thumbnails/", http.FileServer(http.Dir(cfg.Application.Storage.Thumbnails))))

	// Datagrid library assets (Embedded in library)
	sub, _ := fs.Sub(datagrid.UIAssets, "ui/static")
	http.Handle("/ui/static/", http.StripPrefix("/ui/static/", http.FileServer(http.FS(sub))))

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/dashboard", AuthMiddleware(handleDashboard))
	http.HandleFunc("/upload", AuthMiddleware(handleUpload))
	http.HandleFunc("/templates", AuthMiddleware(dgHandler.ServeHTTP))
	http.HandleFunc("/meta", AuthMiddleware(handleMetaPage))
	http.HandleFunc("/resource", AuthMiddleware(handleResourcePage))
	http.HandleFunc("/resource/list", AuthMiddleware(handleResourceList))
	http.HandleFunc("/selection", AuthMiddleware(handleSelection))
	http.HandleFunc("/collect", AuthMiddleware(handleCollect))
	http.HandleFunc("/analyze", AuthMiddleware(handleAnalyze))
	http.HandleFunc("/generator", AuthMiddleware(handleGenerator))
	http.HandleFunc("/detector", AuthMiddleware(handleDetector))
	http.HandleFunc("/detector/save", AuthMiddleware(handleSaveDiscovery))
	http.HandleFunc("/about", AuthMiddleware(handleAbout))
	http.HandleFunc("/ai-tester", AuthMiddleware(handleAITester))
	http.HandleFunc("/ai-tester/chat", AuthMiddleware(handleAIChat))
	http.HandleFunc("/docs/toc", AuthMiddleware(handleDocsTOC))
	http.HandleFunc("/docs/content", AuthMiddleware(handleDocsContent))
	http.HandleFunc("/docs/download", AuthMiddleware(handleDocsDownload))

	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/login-action", handleLoginAction)
	http.HandleFunc("/logout", handleLogout)
	http.HandleFunc("/set-language", handleSetLanguage)
	http.HandleFunc("/reprocess", AuthMiddleware(handleReprocess))
	http.HandleFunc("/reprocess-file", AuthMiddleware(handleReprocessFile))
	http.HandleFunc("/delete-file", AuthMiddleware(handleDeleteFile))
	http.HandleFunc("/copy-to-stage", AuthMiddleware(handleCopyToStage))
	http.HandleFunc("/reprocess-status", AuthMiddleware(handleReprocessStatus))
	http.HandleFunc("/search", AuthMiddleware(handleSearch))
	http.HandleFunc("/search-settings", AuthMiddleware(handleSearchSettings))
	http.HandleFunc("/recent-files", AuthMiddleware(handleRecentFiles))
	http.HandleFunc("/events/status", AuthMiddleware(handleEventsStatus))
	http.HandleFunc("/editor/comments", AuthMiddleware(handleCommentEditor))
	http.HandleFunc("/editor/save", AuthMiddleware(handleSaveComment))
	http.HandleFunc("/editor/save-slide-text", AuthMiddleware(handleSaveSlideText))

	port := cfg.Application.Port
	if port == 0 {
		port = 8088
	}

	fmt.Printf("SlideForge starting on http://localhost:%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	files, err := database.GetAllPPTX(sqlDB)
	if err != nil {
		log.Printf("Failed to get files: %v", err)
	}
	slideCount, err := database.GetTotalSlideCount(sqlDB)
	if err != nil {
		log.Printf("Failed to get slide count: %v", err)
	}
	insightCount, _ := database.GetAIInsightCount(sqlDB)

	data := getBaseData(r, "Dashboard", "dashboard")
	data["Files"] = files
	data["SlideCount"] = slideCount
	data["InsightCount"] = insightCount

	isProcessing, currentFile, totalQueued, startTime := obs.GetStatus()
	data["IsProcessing"] = isProcessing
	data["CurrentFile"] = currentFile
	data["TotalQueued"] = totalQueued
	if isProcessing {
		data["StartTime"] = startTime.Unix()
	}

	// Load settings
	var simThreshold, wordSimThreshold, aiEnabled float64
	sqlDB.QueryRow("SELECT value FROM search_settings WHERE key = 'similarity_threshold'").Scan(&simThreshold)
	sqlDB.QueryRow("SELECT value FROM search_settings WHERE key = 'word_similarity_threshold'").Scan(&wordSimThreshold)
	sqlDB.QueryRow("SELECT value FROM search_settings WHERE key = 'ai_insights_enabled'").Scan(&aiEnabled)
	data["SimThreshold"] = simThreshold
	data["WordSimThreshold"] = wordSimThreshold
	data["AIInsightsEnabled"] = aiEnabled > 0.5 || (aiEnabled == 0 && cfg.AI.Enabled) // Default to config if missing

	renderTemplate(w, "dashboard.html", data)
}

func handleRecentFiles(w http.ResponseWriter, r *http.Request) {
	files, err := database.GetAllPPTX(sqlDB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := getBaseData(r, "", "")
	data["Files"] = files
	renderTemplate(w, "partials/recent_files.html", data)
}

func handleReprocess(w http.ResponseWriter, r *http.Request) {
	go obs.ReprocessAll()
	w.Header().Set("HX-Trigger", "reprocessStarted")

	lang := i18n.GetLang(r)
	html := fmt.Sprintf(`
		<span class="badge bg-warning animate-pulse" style="margin: 0;">%s</span>
		<i class="fas fa-cog fa-spin text-warning"></i>
	`, i18n.T(lang, "processing_all"))
	w.Write([]byte(html))
}

func handleReprocessStatus(w http.ResponseWriter, r *http.Request) {
	isProcessing, currentFile, totalQueued, startTime := obs.GetStatus()
	lang := i18n.GetLang(r)

	if !isProcessing {
		w.Header().Set("HX-Trigger", "reprocessFinished")
		html := fmt.Sprintf(`
			<div class="stat-value" style="font-size: 1.1rem; color: var(--success-color);">
				<i class="fas fa-check-circle"></i> %s
			</div>
			<button onclick="confirmReprocess('0')" class="btn btn-muted btn-sm">
				<i class="fas fa-sync"></i>
			</button>
		`, i18n.T(lang, "system_ready"))
		w.Write([]byte(html))
		return
	}

	duration := time.Since(startTime).Round(time.Second)
	html := fmt.Sprintf(`
		<span class="badge bg-warning animate-pulse" style="margin: 0;">
			%s (%s - %d left)
			<span id="process-duration">%s</span>
		</span>
		<i class="fas fa-cog fa-spin text-warning"></i>
	`, i18n.T(lang, "processing_all"), currentFile, totalQueued, duration)
	w.Write([]byte(html))
}

func handleReprocessFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("fileID")
	if fileID == "" {
		http.Error(w, "fileID required", http.StatusBadRequest)
		return
	}

	var filename, originalPath string
	err := sqlDB.QueryRow("SELECT filename, original_file_path FROM pptx_files WHERE id = $1", fileID).Scan(&filename, &originalPath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// copy back to stage
	destPath := filepath.Join(cfg.Application.Storage.Stage, filename)
	// physical source path logic
	physicalPath := originalPath
	if !filepath.IsAbs(originalPath) {
		parts := strings.Split(originalPath, "/")
		if len(parts) > 1 {
			category := parts[0]
			rel := filepath.Join(parts[1:]...)
			switch category {
			case "template":
				physicalPath = filepath.Join(cfg.Application.Storage.Template, rel)
			case "stage":
				physicalPath = filepath.Join(cfg.Application.Storage.Stage, rel)
			}
		}
	}

	if physicalPath != destPath {
		err = copyFile(physicalPath, destPath)
		if err != nil {
			log.Printf("Failed to copy file to stage: %v", err)
			http.Error(w, "Failed to copy file", http.StatusInternalServerError)
			return
		}
	}

	// Trigger processing immediately (force=true bypasses AUTO toggle)
	go obs.ProcessFile(destPath, true)

	w.Header().Set("HX-Trigger", "refreshFiles")
	w.WriteHeader(http.StatusOK)
}

func handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fileID := r.URL.Query().Get("fileID")
	if fileID == "" {
		http.Error(w, "fileID required", http.StatusBadRequest)
		return
	}

	_, err := sqlDB.Exec("DELETE FROM pptx_files WHERE id = $1", fileID)
	if err != nil {
		log.Printf("Failed to delete file from DB: %v", err)
		http.Error(w, "Failed to delete", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "refreshFiles")
	w.WriteHeader(http.StatusOK)
}

func handleCopyToStage(w http.ResponseWriter, r *http.Request) {
	handleReprocessFile(w, r)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func handleCommentEditor(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("fileID")
	slideNumStr := r.URL.Query().Get("slide")
	if fileID == "" {
		http.Error(w, "fileID required", http.StatusBadRequest)
		return
	}

	var filename, originalPath string
	err := sqlDB.QueryRow("SELECT filename, original_file_path FROM pptx_files WHERE id = $1", fileID).Scan(&filename, &originalPath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	physicalPath := originalPath
	if !filepath.IsAbs(originalPath) {
		parts := strings.Split(originalPath, "/")
		if len(parts) > 1 {
			category := parts[0]
			rel := filepath.Join(parts[1:]...)
			switch category {
			case "template":
				physicalPath = filepath.Join(cfg.Application.Storage.Template, rel)
			case "stage":
				physicalPath = filepath.Join(cfg.Application.Storage.Stage, rel)
			case "original":
				physicalPath = filepath.Join(cfg.Application.Storage.Original, rel)
			}
		}
	}

	content, err := pptx.ExtractSlideContent(physicalPath)
	if err != nil {
		log.Printf("Extraction failed for %s: %v", physicalPath, err)
		http.Error(w, "Failed to extract comments", http.StatusInternalServerError)
		return
	}

	// Fetch overrides
	rows, err := sqlDB.Query("SELECT slide_number, comment_index, text FROM comment_overrides WHERE pptx_path = $1", originalPath)
	overrides := make(map[string]string)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sn, ci int
			var txt string
			if err := rows.Scan(&sn, &ci, &txt); err == nil {
				key := fmt.Sprintf("%d_%d", sn, ci)
				overrides[key] = txt
			}
		}
	}

	data := getBaseData(r, "Comment Editor", "dashboard")
	data["FileID"] = fileID
	data["Filename"] = filename
	data["Slides"] = content
	data["Overrides"] = overrides

	// Numerical keys for sorted iteration in template if needed
	var slideNums []int
	for n := range content {
		slideNums = append(slideNums, n)
	}
	sort.Ints(slideNums)
	data["SlideNums"] = slideNums

	// HTMX Partial Request
	if r.Header.Get("HX-Request") == "true" && slideNumStr != "" {
		sNum, _ := strconv.Atoi(slideNumStr)
		slide, ok := content[sNum]
		if !ok {
			http.Error(w, "Slide not found", http.StatusNotFound)
			return
		}
		data["SelectedSlide"] = slide
		data["SlideNum"] = sNum
		renderTemplate(w, "partials/comment_slide_detail.html", data)
		return
	}

	renderTemplate(w, "comment_editor.html", data)
}

func handleSaveComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	fileID := r.FormValue("fileID")
	slideNum := r.FormValue("slideNum")
	commentIdx := r.FormValue("commentIdx")
	text := r.FormValue("text")

	if fileID == "" || slideNum == "" || commentIdx == "" {
		http.Error(w, "Missing fields", http.StatusBadRequest)
		return
	}

	var originalPath string
	err := sqlDB.QueryRow("SELECT original_file_path FROM pptx_files WHERE id = $1", fileID).Scan(&originalPath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	_, err = sqlDB.Exec(`
		INSERT INTO comment_overrides (pptx_path, slide_number, comment_index, text)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (pptx_path, slide_number, comment_index) DO UPDATE SET text = $4
	`, originalPath, slideNum, commentIdx, text)

	if err != nil {
		log.Printf("DB Error saving comment: %v", err)
		w.Write([]byte("Error saving override"))
	} else {
		w.Write([]byte("Saved"))
	}
}

func handleSaveSlideText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fileID := r.FormValue("fileID")
	slideNum, _ := strconv.Atoi(r.FormValue("slideNum"))
	shapeIdx, _ := strconv.Atoi(r.FormValue("shapeIdx"))
	text := r.FormValue("text")

	if fileID == "" {
		http.Error(w, "fileID required", http.StatusBadRequest)
		return
	}

	var originalPath string
	err := sqlDB.QueryRow("SELECT original_file_path FROM pptx_files WHERE id = $1", fileID).Scan(&originalPath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Resolve physical path
	physicalPath := originalPath
	if !filepath.IsAbs(originalPath) {
		parts := strings.Split(originalPath, "/")
		if len(parts) > 1 {
			category := parts[0]
			rel := filepath.Join(parts[1:]...)
			switch category {
			case "template":
				physicalPath = filepath.Join(cfg.Application.Storage.Template, rel)
			case "stage":
				physicalPath = filepath.Join(cfg.Application.Storage.Stage, rel)
			case "original":
				physicalPath = filepath.Join(cfg.Application.Storage.Remote, rel)
			}
		}
	}

	err = pptx.UpdateSlideText(physicalPath, slideNum, shapeIdx, text)
	if err != nil {
		log.Printf("Error updating PPTX text: %v", err)
		w.Write([]byte("<span class='badge bg-danger'>Error</span>"))
		return
	}

	w.Write([]byte("<span class='badge bg-success'>Updated</span>"))
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	mode := r.URL.Query().Get("mode") // fts, similarity, word_similarity
	lang := i18n.GetLang(r)

	if query == "" {
		w.Write([]byte("<div class='text-muted'>Enter search query</div>"))
		return
	}

	var threshold float64
	switch mode {
	case "similarity":
		sqlDB.QueryRow("SELECT value FROM search_settings WHERE key = 'similarity_threshold'").Scan(&threshold)
	case "word_similarity":
		sqlDB.QueryRow("SELECT value FROM search_settings WHERE key = 'word_similarity_threshold'").Scan(&threshold)
	}

	var rows *sql.Rows
	var err error

	sqlDB.Exec("SET search_path TO slideforge, public")

	switch mode {
	case "similarity":
		sqlDB.Exec("SELECT set_limit($1)", threshold)
		rows, err = sqlDB.Query(`
			SELECT s.id, s.pptx_file_id, s.slide_number, s.png_path, s.content, f.filename, s.content as snippet, s.title
			FROM collected_slides s
			JOIN pptx_files f ON s.pptx_file_id = f.id
			WHERE s.content % $1
			ORDER BY similarity(s.content, $1) DESC
			LIMIT 20`, query)
	case "word_similarity":
		rows, err = sqlDB.Query(`
			SELECT s.id, s.pptx_file_id, s.slide_number, s.png_path, s.content, f.filename, s.content as snippet, s.title
			FROM collected_slides s
			JOIN pptx_files f ON s.pptx_file_id = f.id
			WHERE s.content %> $1
			ORDER BY s.content <<-> $1
			LIMIT 20`, query)
	default: // FTS
		ftsCol := "fts_combined"
		config := "english"
		switch lang {
		case "hu":
			config = "hungarian"
			ftsCol = "fts_hu"
		case "en":
			ftsCol = "fts_en"
		}

		rows, err = sqlDB.Query(fmt.Sprintf(`
			SELECT s.id, s.pptx_file_id, s.slide_number, s.png_path, s.content, f.filename,
			       ts_headline('%s', s.content || ' ' || s.comments, websearch_to_tsquery('%s', $1), 'StartSel=<mark>, StopSel=</mark>, MaxWords=35, MinWords=15') as snippet, s.title
			FROM collected_slides s
			JOIN pptx_files f ON s.pptx_file_id = f.id
			WHERE s.%s @@ websearch_to_tsquery('%s', $1) OR f.%s @@ websearch_to_tsquery('%s', $1)
			ORDER BY ts_rank_cd(s.%s, websearch_to_tsquery('%s', $1)) * 0.4 + 
			         ts_rank_cd(f.%s, websearch_to_tsquery('%s', $1)) * 0.6 DESC
			LIMIT 20`, config, config, ftsCol, config, ftsCol, config, ftsCol, config, ftsCol, config), query)
	}

	if err != nil {
		log.Printf("Search error: %v", err)
		w.Write([]byte(fmt.Sprintf("<div class='alert alert-danger'>Search error: %v</div>", err)))
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, fileID, slideNum int
		var pngPath, content, filename, snippet, title string
		rows.Scan(&id, &fileID, &slideNum, &pngPath, &content, &filename, &snippet, &title)
		results = append(results, map[string]interface{}{
			"ID":          id,
			"FileID":      fileID,
			"SlideNumber": slideNum,
			"PNGPath":     pngPath,
			"Content":     content,
			"Filename":    filename,
			"Snippet":     template.HTML(snippet),
			"Title":       title,
		})
	}

	data := map[string]interface{}{
		"Results": results,
		"Query":   query,
	}

	// Small inline template for results
	resTmpl := `
	<div class='search-results-grid'>
		{{range .Results}}
		<div class='search-result-card' onclick="window.location='/selection?fileID={{.FileID}}'">
			<img src='/thumbnails/{{.PNGPath}}' loading='lazy'>
			<div class='result-info'>
				<strong>{{stripExt .Filename}}</strong> - {{ if .Title }}{{ .Title }}{{ else }}Slide {{.SlideNumber}}{{ end }}
				<p class='content-snippet'>{{.Snippet}}</p>
			</div>
		</div>
		{{else}}
		<div class='text-center p-4'>No results found.</div>
		{{end}}
	</div>`

	t, _ := template.New("results").Funcs(template.FuncMap{
		"stripExt": func(filename string) string {
			return strings.TrimSuffix(filename, filepath.Ext(filename))
		},
	}).Parse(resTmpl)
	t.Execute(w, data)
}

func handleSearchSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	key := r.FormValue("key")
	val := r.FormValue("value")
	_, err := sqlDB.Exec("INSERT INTO search_settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2", key, val)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error saving setting"))
	} else {
		w.Write([]byte("OK"))
	}
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	t := template.Must(tmpl.Clone())
	// Try to find the specific template in embedded FS
	_, err := t.ParseFS(assets.EmbeddedAssets, "ui/templates/"+name)
	if err != nil {
		log.Printf("Template parse error (%s): %v", name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = t.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("Template execution error (%s): %v", name, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

	destPath := filepath.Join(cfg.Application.Storage.Stage, header.Filename)
	dest, err := os.Create(destPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dest.Close()
	io.Copy(dest, file)

	// Process PPTX to PNGs
	thumbDir := filepath.Join(cfg.Application.Storage.Thumbnails, header.Filename)
	pngFiles, err := pptx.ExtractSlidesToPNG(destPath, thumbDir, cfg.Application.Storage.Temp)
	if err != nil {
		log.Printf("PNG extraction failed: %v", err)
	}

	// Extract Slide Content (Text & Styles)
	slideDataMap, err := pptx.ExtractSlideContent(destPath)
	if err != nil {
		log.Printf("Failed to extract slide content: %v", err)
	}

	// Insert File into DB
	fileID, err := database.SavePPTXMetadata(sqlDB, &database.PPTXFile{
		Filename:         header.Filename,
		OriginalFilePath: destPath,
		ThumbnailDirPath: thumbDir,
		Metadata:         []byte("{}"),
		AISummary:        "", // Will be updated later
	})
	if err != nil {
		log.Printf("DB insert failed: %v", err)
	}

	var slideSummaries []string
	ctx := r.Context()

	// Check if AI is enabled globally and in settings
	var aiEnabledVal float64
	sqlDB.QueryRow("SELECT value FROM search_settings WHERE key = 'ai_insights_enabled'").Scan(&aiEnabledVal)
	aiEnabled := aiEnabledVal > 0.5 || (aiEnabledVal == 0 && cfg.AI.Enabled)

	// Save individual slides
	for i, png := range pngFiles {
		slideNum := i + 1
		content := ""
		styleJSON := []byte("{}")
		slideSummary := ""
		slideTitle := fmt.Sprintf("Slide %d", slideNum) // Default

		if data, ok := slideDataMap[slideNum]; ok {
			content = data.Text
			if sj, err := json.Marshal(data.Styles); err == nil {
				styleJSON = sj
			}

			// Generate slide summary & title
			if content != "" && aiEnabled {
				// Summary
				result, err := aiClient.SummarizeText(ctx, content)
				if err == nil {
					slideSummary = result.Content
					slideSummaries = append(slideSummaries, result.Content)
					// Log usage
					database.LogAIUsage(sqlDB, &database.AIUsage{
						Provider:         cfg.AI.ActiveProvider,
						Model:            cfg.AI.Providers[cfg.AI.ActiveProvider].Model,
						PromptTokens:     result.Usage.PromptTokens,
						CompletionTokens: result.Usage.CompletionTokens,
						TotalTokens:      result.Usage.TotalTokens,
						Cost:             result.Cost,
					})
				}

				// Title
				rawTitleResult, err := aiClient.ExtractSlideTitle(ctx, content)
				if err == nil && rawTitleResult.Content != "" {
					slideTitle = fmt.Sprintf("%d. %s", slideNum, rawTitleResult.Content)
					// Log usage
					database.LogAIUsage(sqlDB, &database.AIUsage{
						Provider:         cfg.AI.ActiveProvider,
						Model:            cfg.AI.Providers[cfg.AI.ActiveProvider].Model,
						PromptTokens:     rawTitleResult.Usage.PromptTokens,
						CompletionTokens: rawTitleResult.Usage.CompletionTokens,
						TotalTokens:      rawTitleResult.Usage.TotalTokens,
						Cost:             rawTitleResult.Cost,
					})
				}
			}
		}

		// The `png` variable already contains the full path relative to the web root, e.g., "thumbnails/filename/slide_X.png"
		// So, we just need to ensure it's slash-separated.
		// Ensure all thumbnail paths start with 'thumbnails/'
		relPNGPath, _ := filepath.Rel(thumbDir, png)
		thumbSubPath := filepath.Base(thumbDir) // This is header.Filename
		err = database.SaveSlide(sqlDB, &database.Slide{
			PPTXFileID: fileID,
			SlideNum:   slideNum,
			PNGPath:    filepath.ToSlash(filepath.Join(thumbSubPath, relPNGPath)),
			Content:    content,
			StyleInfo:  styleJSON,
			AIAnalysis: []byte("{}"),
			AISummary:  slideSummary,
			Title:      slideTitle,
		})
		if err != nil {
			log.Printf("Failed to save slide %d: %v", slideNum, err)
		}
	}

	// Generate overall summary & title
	if len(slideSummaries) > 0 && aiEnabled {
		// Summary
		fullTextForSummary := strings.Join(slideSummaries, "\n")
		overallSummaryResult, err := aiClient.SummarizeText(ctx, "Provide a concise summary of this presentation based on its slide summaries: \n"+fullTextForSummary)
		if err == nil {
			database.UpdatePPTXSummary(sqlDB, fileID, overallSummaryResult.Content)
			// Log usage
			database.LogAIUsage(sqlDB, &database.AIUsage{
				Provider:         cfg.AI.ActiveProvider,
				Model:            cfg.AI.Providers[cfg.AI.ActiveProvider].Model,
				PromptTokens:     overallSummaryResult.Usage.PromptTokens,
				CompletionTokens: overallSummaryResult.Usage.CompletionTokens,
				TotalTokens:      overallSummaryResult.Usage.TotalTokens,
				Cost:             overallSummaryResult.Cost,
			})
		}

		// Title (from first slide)
		if data, ok := slideDataMap[1]; ok && data.Text != "" {
			titleResult, err := aiClient.ExtractTitle(ctx, data.Text)
			if err == nil && titleResult.Content != "" {
				database.UpdatePPTXTitle(sqlDB, fileID, titleResult.Content)
				// Log usage
				database.LogAIUsage(sqlDB, &database.AIUsage{
					Provider:         cfg.AI.ActiveProvider,
					Model:            cfg.AI.Providers[cfg.AI.ActiveProvider].Model,
					PromptTokens:     titleResult.Usage.PromptTokens,
					CompletionTokens: titleResult.Usage.CompletionTokens,
					TotalTokens:      titleResult.Usage.TotalTokens,
					Cost:             titleResult.Cost,
				})
			}
		}
	}

	// Redirect to selection view for this file
	http.Redirect(w, r, fmt.Sprintf("/selection?fileID=%d", fileID), http.StatusSeeOther)
}

func handleSelection(w http.ResponseWriter, r *http.Request) {
	fileIDStr := r.URL.Query().Get("fileID")
	if fileIDStr == "" {
		fileIDStr = r.URL.Query().Get("id")
	}
	var fileID int
	fmt.Sscanf(fileIDStr, "%d", &fileID)

	slides, err := database.GetSlidesByFile(sqlDB, fileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch PPTX title/filename for header
	var pptxTitle, filename string
	err = sqlDB.QueryRow("SELECT title, filename FROM pptx_files WHERE id = $1", fileID).Scan(&pptxTitle, &filename)
	if err != nil {
		pptxTitle = "Slide Selection"
	} else if pptxTitle == "" {
		pptxTitle = filename
	}

	data := getBaseData(r, pptxTitle, "dashboard")
	data["slides"] = slides
	data["FileID"] = fileID
	data["Filename"] = filename

	// Fetch PPTX summary
	var pptxSummary string
	sqlDB.QueryRow("SELECT ai_summary FROM pptx_files WHERE id = $1", fileID).Scan(&pptxSummary)
	data["pptxSummary"] = pptxSummary

	renderTemplate(w, "selection.html", data)
}

func handleCollect(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Successfully collected slides from file %s!", r.FormValue("fileID"))
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	fileIDStr := r.FormValue("fileID")
	selectedSlides := r.Form["selectedSlides"]

	// Check if AI is enabled
	var aiEnabledVal float64
	sqlDB.QueryRow("SELECT value FROM search_settings WHERE key = 'ai_insights_enabled'").Scan(&aiEnabledVal)
	aiEnabled := aiEnabledVal > 0.5 || (aiEnabledVal == 0 && cfg.AI.Enabled)

	if !aiEnabled {
		w.Write([]byte("<p class='text-warning'>AI operations are currently disabled. Please turn AI ON in the header to use this feature.</p>"))
		return
	}

	if len(selectedSlides) == 0 {
		w.Write([]byte("<p class='text-warning'>No slides selected for analysis.</p>"))
		return
	}

	var fileID int
	fmt.Sscanf(fileIDStr, "%d", &fileID)

	slides, err := database.GetSlidesByFile(sqlDB, fileID)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("<p class='text-danger'>Error fetching slides: %v</p>", err)))
		return
	}

	var selectedPath string
	for _, s := range slides {
		for _, selNum := range selectedSlides {
			var num int
			fmt.Sscanf(selNum, "%d", &num)
			if s.SlideNum == num {
				selectedPath = s.PNGPath
				break
			}
		}
		if selectedPath != "" {
			break
		}
	}

	if selectedPath == "" {
		w.Write([]byte("<p class='text-danger'>Error finding selected slide.</p>"))
		return
	}

	analysisResult, err := aiClient.AnalyzeSlide(r.Context(), strings.TrimPrefix(selectedPath, "/"))
	if err != nil {
		w.Write([]byte(fmt.Sprintf("<p class='text-danger'>AI Analysis failed: %v</p>", err)))
		return
	}

	// Log usage
	database.LogAIUsage(sqlDB, &database.AIUsage{
		Provider:         cfg.AI.ActiveProvider,
		Model:            cfg.AI.Providers[cfg.AI.ActiveProvider].Model,
		PromptTokens:     analysisResult.Usage.PromptTokens,
		CompletionTokens: analysisResult.Usage.CompletionTokens,
		TotalTokens:      analysisResult.Usage.TotalTokens,
		Cost:             analysisResult.Cost,
	})

	w.Write([]byte(fmt.Sprintf("<div class='animate-fade-in'>%s</div>", analysisResult.Content)))
}

type FileNode struct {
	Name     string
	Files    []database.PPTXWithSlides
	Children map[string]*FileNode
}

func buildFileTree(files []database.PPTXWithSlides) *FileNode {
	root := &FileNode{Name: "Root", Children: make(map[string]*FileNode)}

	for _, f := range files {
		// Use original_file_path to determine hierarchy
		path := f.OriginalFilePath
		// Try to make it relative to some common roots if possible,
		// but for now let's just use the path parts.
		// We'll strip the common prefix if we can identify it.

		// For SlideForge, let's try to strip the absolute prefix until /mnt/bdo/ or just use the last few segments
		parts := strings.Split(filepath.Dir(path), string(os.PathSeparator))

		currentNode := root
		for _, part := range parts {
			if part == "" || part == "home" || part == "gnemet" || part == "GitHub" || part == "SlideForge" || part == "mnt" || part == "bdo" {
				continue
			}
			if _, ok := currentNode.Children[part]; !ok {
				currentNode.Children[part] = &FileNode{Name: part, Children: make(map[string]*FileNode)}
			}
			currentNode = currentNode.Children[part]
		}
		currentNode.Files = append(currentNode.Files, f)
	}
	return root
}

func handleGenerator(w http.ResponseWriter, r *http.Request) {
	files, err := database.GetAllPPTXWithSlides(sqlDB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tree := buildFileTree(files)

	data := getBaseData(r, "Slide Generator", "generator")
	data["Files"] = files
	data["FileTree"] = tree
	renderTemplate(w, "generator.html", data)
}

func handleDetector(w http.ResponseWriter, r *http.Request) {
	files, err := database.GetAllPPTXWithSlides(sqlDB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tree := buildFileTree(files)
	discovered, _ := database.GetDiscoveredPlaceholders(sqlDB)
	discoveredJSON, _ := json.Marshal(discovered)

	data := getBaseData(r, "Placeholder Detector", "detector")
	data["Files"] = files
	data["FileTree"] = tree
	data["Discovered"] = template.HTML(discoveredJSON)
	renderTemplate(w, "detector.html", data)
}

func handleSaveDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fileID, _ := strconv.Atoi(r.FormValue("fileID"))
	slideNum, _ := strconv.Atoi(r.FormValue("slideNum"))
	placeholder := r.FormValue("placeholder")
	key := r.FormValue("key")

	if fileID == 0 || slideNum == 0 || placeholder == "" || key == "" {
		http.Error(w, "Missing fields", http.StatusBadRequest)
		return
	}

	dp := &database.DiscoveredPlaceholder{
		PPTXFileID:      fileID,
		SlideNumber:     slideNum,
		PlaceholderText: placeholder,
		MetadataKey:     key,
	}

	err := database.SaveDiscoveredPlaceholder(sqlDB, dp)
	if err != nil {
		log.Printf("DB Error saving discovery: %v", err)
		w.Header().Set("HX-Trigger", "discoveryError")
		w.Write([]byte("Error saving"))
	} else {
		w.Header().Set("HX-Trigger", "discoverySaved")
		w.Write([]byte("Saved successfully"))
	}
}

func handleAbout(w http.ResponseWriter, r *http.Request) {
	data := getBaseData(r, "About", "about")
	renderTemplate(w, "about.html", data)
}

func handleAITester(w http.ResponseWriter, r *http.Request) {
	data := getBaseData(r, "AI Tester", "ai_tester")

	// Placeholder for Billing/Usage Data
	data["Usage"] = map[string]interface{}{
		"Balance": "$0.42",
		"Budget":  "$10.00",
		"Status":  "active",
	}

	renderTemplate(w, "ai_chat.html", data)
}

func handleAIChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	prompt := r.FormValue("prompt")
	if prompt == "" {
		return
	}

	// 1. Render User Message snippet
	userHtml := fmt.Sprintf(`<div class="message user">%s</div>`, template.HTMLEscapeString(prompt))

	// 2. Call AI
	// Check if AI is enabled
	var aiEnabledVal float64
	sqlDB.QueryRow("SELECT value FROM search_settings WHERE key = 'ai_insights_enabled'").Scan(&aiEnabledVal)
	aiEnabled := aiEnabledVal > 0.5 || (aiEnabledVal == 0 && cfg.AI.Enabled)

	if !aiEnabled {
		w.Write([]byte(userHtml + `<div class="message ai">AI operations are currently disabled. Please turn AI ON in the header.</div>`))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	result, err := aiClient.GenerateContent(ctx, ai.PresentationArchitect, prompt)
	var response string
	if err != nil {
		response = fmt.Sprintf("Error: %v", err)
	} else {
		response = result.Content
		// Log usage
		database.LogAIUsage(sqlDB, &database.AIUsage{
			Provider:         cfg.AI.ActiveProvider,
			Model:            cfg.AI.Providers[cfg.AI.ActiveProvider].Model,
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
			Cost:             result.Cost,
		})
	}

	// 3. Render AI Response snippet
	aiMarkdown := blackfriday.Run([]byte(response))
	aiHtml := fmt.Sprintf(`<div class="message ai">%s</div>`, string(aiMarkdown))

	// Return both
	w.Write([]byte(userHtml + aiHtml))
}

func handleDocsTOC(w http.ResponseWriter, r *http.Request) {
	var html strings.Builder
	html.WriteString("<div id='docs-toc-list'>")

	filepath.Walk("docs", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			relPath, _ := filepath.Rel("docs", path)
			name := strings.TrimSuffix(info.Name(), ".md")
			name = strings.Title(strings.ReplaceAll(name, "_", " "))

			// Indent based on depth
			depth := strings.Count(relPath, string(os.PathSeparator))
			padding := depth * 15

			html.WriteString(fmt.Sprintf("<a hx-get='/docs/content?file=%s' hx-target='#docs-content-area' style='padding-left: %dpx;'>%s</a>", relPath, 20+padding, name))
		}
		return nil
	})

	html.WriteString("</div>")
	w.Write([]byte(html.String()))
}

func handleDocsDownload(w http.ResponseWriter, r *http.Request) {
	// For now, look for a pre-generated PDF or return a polite message
	pdfPath := fmt.Sprintf("%s_v%s.pdf", cfg.Application.Name, cfg.Application.Version)
	if _, err := os.Stat(pdfPath); err == nil {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", pdfPath))
		w.Header().Set("Content-Type", "application/pdf")
		http.ServeFile(w, r, pdfPath)
		return
	}

	http.Error(w, "Documentation PDF not yet generated for this version. Please run 'go run scripts/gen_pdf_docs.go' first.", http.StatusNotFound)
}

func handleDocsContent(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("file")
	if fileName == "" {
		fileName = "workflow.md"
	}

	path := filepath.Join("docs", fileName)
	content, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	output := blackfriday.Run(content)
	w.Write([]byte(fmt.Sprintf("<div class='docs-content animate-fade-in'>%s</div>", output)))
}

// Helpers & Auth

func getBaseData(r *http.Request, title string, activeNav string) map[string]interface{} {
	lang := i18n.GetLang(r)
	availableLangs := i18n.GetAvailableLangs()
	langsJSON, _ := json.Marshal(availableLangs)

	authUser := ""
	if cookie, err := r.Cookie("auth_user"); err == nil {
		authUser = cookie.Value
	}

	userName := "Guest"
	if cookie, err := r.Cookie("user_name"); err == nil {
		userName = cookie.Value
	}

	appName := "SlideForge"
	if cfg != nil && cfg.Application.Name != "" {
		appName = cfg.Application.Name
	}
	version := "v1.0.0"
	if cfg != nil && cfg.Application.Version != "" {
		version = cfg.Application.Version
	}
	author := "Unknown"
	if cfg != nil && cfg.Application.Author != "" {
		author = cfg.Application.Author
	}
	lastBuild := time.Now().Format("2006-01-02 15:04") // Default or from config if set
	if cfg != nil && cfg.Application.LastBuild != "" {
		lastBuild = cfg.Application.LastBuild
	}
	engine := "Standard Engine"
	if cfg != nil && cfg.Application.Engine != "" {
		engine = cfg.Application.Engine
	}
	copyright := fmt.Sprintf("%d", time.Now().Year())
	if cfg != nil && cfg.Application.Copyright != "" {
		copyright = cfg.Application.Copyright
	}
	aiProvider := "Unknown"
	if cfg != nil {
		aiProvider = cfg.AI.ActiveProvider
	}
	dbName := "Unknown"
	if cfg != nil {
		dbName = cfg.Database.DBName
	}

	allTimeCost, _ := database.GetTotalAICost(sqlDB)

	isProcessing, _, _, _ := obs.GetStatus()

	var aiEnabledVal float64
	sqlDB.QueryRow("SELECT value FROM search_settings WHERE key = 'ai_insights_enabled'").Scan(&aiEnabledVal)
	aiInsightsEnabled := aiEnabledVal > 0.5 || (aiEnabledVal == 0 && cfg.AI.Enabled)

	var autoProcessVal float64
	sqlDB.QueryRow("SELECT value FROM search_settings WHERE key = 'auto_process_enabled'").Scan(&autoProcessVal)
	autoProcessEnabled := autoProcessVal > 0.5 || (autoProcessVal == 0 && true) // Default to ON if not set

	aiStatus := "Off"
	if aiInsightsEnabled {
		aiStatus = "Unaccess"
		activeProvider := cfg.AI.Providers[cfg.AI.ActiveProvider]
		if activeProvider.Key != "" || activeProvider.Driver == "mock" {
			aiStatus = "Live"
		}
	}

	return map[string]interface{}{
		"Title":              title,
		"ActiveNav":          activeNav,
		"Lang":               lang,
		"LangsJSON":          string(langsJSON),
		"IsAuth":             authUser != "",
		"UserName":           userName,
		"IsProcessing":       isProcessing,
		"AllTimeAICost":      allTimeCost,
		"AIInsightsEnabled":  aiInsightsEnabled,
		"AutoProcessEnabled": autoProcessEnabled,
		"AIStatus":           aiStatus,
		"AIStatusClass":      strings.ToLower(aiStatus),
		"LDAPEnabled":        cfg.Ldap.Enabled,
		"AuthEnabled":        cfg.Application.Authentication,
		"App": map[string]string{
			"Name":       appName,
			"Version":    version,
			"Author":     author,
			"LastBuild":  lastBuild,
			"Engine":     engine,
			"Copyright":  copyright,
			"AIProvider": aiProvider,
			"DBName":     dbName,
		},
	}
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg != nil && (cfg.Application.AuthType == "none" || !cfg.Application.Authentication) {
			next(w, r)
			return
		}
		cookie, err := r.Cookie("auth_user")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	data := getBaseData(r, "Login", "")
	if err := r.URL.Query().Get("error"); err != "" {
		data["Error"] = err
	}
	renderTemplate(w, "login.html", data)
}

func handleLoginAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	// Dummy auth: Any non-empty username works
	if username == "" {
		http.Redirect(w, r, "/login?error=Username required", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_user",
		Value:    username,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "user_name",
		Value:    username,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: false,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:    "auth_user",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
	})
	http.SetCookie(w, &http.Cookie{
		Name:    "user_name",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func handleSetLanguage(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang != "" {
		http.SetCookie(w, &http.Cookie{
			Name:    "lang",
			Value:   lang,
			Path:    "/",
			Expires: time.Now().Add(365 * 24 * time.Hour),
		})
	}
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func processLogs() {
	for msg := range logChan {
		// Append to buffer
		logMutex.Lock()
		logBuffer = append(logBuffer, msg)
		if len(logBuffer) > 50 {
			logBuffer = logBuffer[1:]
		}
		logMutex.Unlock()

		// Broadcast to SSE clients
		sseMutex.Lock()
		for clientChan := range sseClients {
			select {
			case clientChan <- msg:
			default:
				// Slow client, skip
			}
		}
		sseMutex.Unlock()
	}
}

func handleEventsStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientChan := make(chan string, 10)
	sseMutex.Lock()
	sseClients[clientChan] = true
	sseMutex.Unlock()

	defer func() {
		sseMutex.Lock()
		delete(sseClients, clientChan)
		sseMutex.Unlock()
		close(clientChan)
	}()

	flusher, _ := w.(http.Flusher)

	// Stream updates
	ctx := r.Context()
	for {
		select {
		case logMsg := <-clientChan:
			isProcessing, currentFile, totalQueued, startTime := obs.GetStatus()
			data, _ := json.Marshal(map[string]interface{}{
				"is_processing": isProcessing,
				"current_file":  currentFile,
				"total_queued":  totalQueued,
				"start_time":    startTime.Unix(),
				"last_log":      logMsg,
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			if flusher != nil {
				flusher.Flush()
			}
		case <-time.After(3 * time.Second):
			// Heartbeat & Status Update (even if no logs)
			isProcessing, currentFile, totalQueued, startTime := obs.GetStatus()
			data, _ := json.Marshal(map[string]interface{}{
				"is_processing": isProcessing,
				"current_file":  currentFile,
				"total_queued":  totalQueued,
				"start_time":    startTime.Unix(),
				"last_log":      "",
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			if flusher != nil {
				flusher.Flush()
			}
		case <-ctx.Done():
			return
		}
	}
}

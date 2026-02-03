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
	"strings"

	"encoding/json"
	"sync"
	"time"

	"github.com/gnemet/SlideForge/internal/ai"
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
	os.MkdirAll("uploads", 0755)
	os.MkdirAll("thumbnails", 0755)

	i18n.Init()

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
	}
	// Merge Datagrid Template Funcs (renderRow, etc.)
	for k, v := range datagrid.TemplateFuncs() {
		funcMap[k] = v
	}

	tmpl = template.Must(template.New("").Funcs(funcMap).ParseGlob("ui/templates/layout/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("ui/templates/partials/*.html"))
	// Automagically include all datagrid library templates
	tmpl = template.Must(tmpl.ParseFS(datagrid.UIAssets, "ui/templates/partials/datagrid/*.html"))

	// Datagrid Handler for PPTX Files
	dgHandler := datagrid.NewHandler(sqlDB, "pptx_files", []datagrid.UIColumn{
		{Field: "filename", Label: "Name", Visible: true, Sortable: true},
		{Field: "created_at", Label: "Uploaded", Visible: true, Sortable: true},
		{Field: "is_template", Label: "Template", Visible: true, Type: "boolean"},
	}, datagrid.DatagridConfig{})

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("ui/static"))))
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
	http.HandleFunc("/about", AuthMiddleware(handleAbout))
	http.HandleFunc("/docs/toc", AuthMiddleware(handleDocsTOC))
	http.HandleFunc("/docs/content", AuthMiddleware(handleDocsContent))

	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/login-action", handleLoginAction)
	http.HandleFunc("/logout", handleLogout)
	http.HandleFunc("/set-language", handleSetLanguage)
	http.HandleFunc("/reprocess", AuthMiddleware(handleReprocess))
	http.HandleFunc("/reprocess-status", AuthMiddleware(handleReprocessStatus))
	http.HandleFunc("/search", AuthMiddleware(handleSearch))
	http.HandleFunc("/search-settings", AuthMiddleware(handleSearchSettings))
	http.HandleFunc("/events/logs", AuthMiddleware(handleEventsLogs))

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

	data := getBaseData(r, "Dashboard", "dashboard")
	data["Files"] = files
	data["SlideCount"] = slideCount
	data["IsProcessing"] = obs.IsProcessing()

	// Load settings
	var simThreshold, wordSimThreshold float64
	sqlDB.QueryRow("SELECT value FROM search_settings WHERE key = 'similarity_threshold'").Scan(&simThreshold)
	sqlDB.QueryRow("SELECT value FROM search_settings WHERE key = 'word_similarity_threshold'").Scan(&wordSimThreshold)
	data["SimThreshold"] = simThreshold
	data["WordSimThreshold"] = wordSimThreshold

	renderTemplate(w, "dashboard.html", data)
}

func handleReprocess(w http.ResponseWriter, r *http.Request) {
	go obs.ReprocessAll()
	w.Header().Set("HX-Trigger", "reprocessStarted")
	w.Write([]byte("Processing started..."))
}

func handleReprocessStatus(w http.ResponseWriter, r *http.Request) {
	lang := i18n.GetLang(r)
	if obs.IsProcessing() {
		w.Write([]byte(fmt.Sprintf("<span class='badge bg-warning animate-pulse'>%s...</span>", i18n.T(lang, "processing"))))
	} else {
		btnHtml := fmt.Sprintf("<button hx-post='/reprocess' hx-target='#reprocess-status' class='btn btn-primary btn-sm'><i class='fas fa-sync'></i> %s</button>", i18n.T(lang, "reprocess_all_pptx"))
		w.Write([]byte(btnHtml))
	}
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
			       ts_headline('%s', s.content, websearch_to_tsquery('%s', $1), 'StartSel=<mark>, StopSel=</mark>, MaxWords=35, MinWords=15') as snippet, s.title
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
			<img src='{{.PNGPath}}' loading='lazy'>
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
		w.Write([]byte("Error saving setting"))
	} else {
		w.Write([]byte("Saved " + val))
	}
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	t := template.Must(tmpl.Clone())
	t = template.Must(t.ParseFiles("ui/templates/" + name))
	err := t.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("Template error (%s): %v", name, err)
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
	pngFiles, err := pptx.ExtractSlidesToPNG(destPath, thumbDir)
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
			if content != "" {
				// Summary
				summary, err := aiClient.SummarizeText(ctx, content)
				if err == nil {
					slideSummary = summary
					slideSummaries = append(slideSummaries, summary)
				}

				// Title
				rawTitle, err := aiClient.ExtractSlideTitle(ctx, content)
				if err == nil && rawTitle != "" {
					slideTitle = fmt.Sprintf("%d. %s", slideNum, rawTitle)
				}
			}
		}

		err = database.SaveSlide(sqlDB, &database.Slide{
			PPTXFileID: fileID,
			SlideNum:   slideNum,
			PNGPath:    "/" + png, // Web accessible path
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
	if len(slideSummaries) > 0 {
		// Summary
		fullTextForSummary := strings.Join(slideSummaries, "\n")
		overallSummary, err := aiClient.SummarizeText(ctx, "Provide a concise summary of this presentation based on its slide summaries: \n"+fullTextForSummary)
		if err == nil {
			database.UpdatePPTXSummary(sqlDB, fileID, overallSummary)
		}

		// Title (from first slide)
		if data, ok := slideDataMap[1]; ok && data.Text != "" {
			title, err := aiClient.ExtractTitle(ctx, data.Text)
			if err == nil && title != "" {
				database.UpdatePPTXTitle(sqlDB, fileID, title)
			}
		}
	}

	// Redirect to selection view for this file
	http.Redirect(w, r, fmt.Sprintf("/selection?fileID=%d", fileID), http.StatusSeeOther)
}

func handleSelection(w http.ResponseWriter, r *http.Request) {
	fileIDStr := r.URL.Query().Get("fileID")
	var fileID int
	fmt.Sscanf(fileIDStr, "%d", &fileID)

	slides, err := database.GetSlidesByFile(sqlDB, fileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := getBaseData(r, "Slide Selection", "dashboard")
	data["slides"] = slides
	data["FileID"] = fileID

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

	analysis, err := aiClient.AnalyzeSlide(r.Context(), strings.TrimPrefix(selectedPath, "/"))
	if err != nil {
		w.Write([]byte(fmt.Sprintf("<p class='text-danger'>AI Analysis failed: %v</p>", err)))
		return
	}

	w.Write([]byte(fmt.Sprintf("<div class='animate-fade-in'>%s</div>", analysis)))
}

func handleGenerator(w http.ResponseWriter, r *http.Request) {
	files, err := database.GetAllPPTXWithSlides(sqlDB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := getBaseData(r, "Slide Generator", "generator")
	data["Files"] = files
	renderTemplate(w, "generator.html", data)
}

func handleAbout(w http.ResponseWriter, r *http.Request) {
	data := getBaseData(r, "About", "about")
	renderTemplate(w, "about.html", data)
}

func handleDocsTOC(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir("docs")
	if err != nil {
		http.Error(w, "Failed to read docs", http.StatusInternalServerError)
		return
	}

	var html strings.Builder
	html.WriteString("<div id='docs-toc-list'>")
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
			name := strings.TrimSuffix(f.Name(), ".md")
			name = strings.Title(strings.ReplaceAll(name, "_", " "))
			html.WriteString(fmt.Sprintf("<a hx-get='/docs/content?file=%s' hx-target='#docs-content-area'>%s</a>", f.Name(), name))
		}
	}
	html.WriteString("</div>")
	w.Write([]byte(html.String()))
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

	return map[string]interface{}{
		"Title":        title,
		"ActiveNav":    activeNav,
		"Lang":         lang,
		"LangsJSON":    string(langsJSON),
		"IsAuth":       authUser != "",
		"UserName":     userName,
		"IsProcessing": obs.IsProcessing(),
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

func handleEventsLogs(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client channel
	clientChan := make(chan string, 10)

	// Register client
	sseMutex.Lock()
	sseClients[clientChan] = true
	sseMutex.Unlock()

	defer func() {
		sseMutex.Lock()
		delete(sseClients, clientChan)
		sseMutex.Unlock()
		close(clientChan)
	}()

	// Send history
	logMutex.Lock()
	for _, msg := range logBuffer {
		fmt.Fprintf(w, "data: %s\n\n", msg)
	}
	logMutex.Unlock()
	w.(http.Flusher).Flush()

	// Stream new messages
	ctx := r.Context()
	for {
		select {
		case msg := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			w.(http.Flusher).Flush()
		case <-ctx.Done():
			return
		}
	}
}

package observer

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"io"

	"github.com/fsnotify/fsnotify"
	"github.com/gnemet/SlideForge/internal/ai"
	"github.com/gnemet/SlideForge/internal/config"
	"github.com/gnemet/SlideForge/internal/database"
	"github.com/gnemet/SlideForge/internal/pptx"
)

type Observer struct {
	cfg         *config.Config
	db          *sql.DB
	aiClient    *ai.Client
	activeTasks int
	currentFile string
	totalQueued int
	startTime   time.Time
	mu          sync.Mutex
	LogChan     chan string
}

func NewObserver(cfg *config.Config, db *sql.DB, ai *ai.Client, logChan chan string) *Observer {
	return &Observer{
		cfg:      cfg,
		db:       db,
		aiClient: ai,
		LogChan:  logChan,
	}
}

func (o *Observer) log(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Println(msg)
	if o.LogChan != nil {
		select {
		case o.LogChan <- msg:
		default:
			// fast non-blocking drop if buffer full
		}
	}
}

func (o *Observer) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Watch Stage directory
	stageDir := o.cfg.Application.Storage.Stage
	if stageDir == "" {
		return fmt.Errorf("stage storage directory not configured")
	}

	// Ensure directories exist
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return fmt.Errorf("failed to create stage directory: %v", err)
	}

	templateDir := o.cfg.Application.Storage.Template
	if templateDir != "" {
		if err := os.MkdirAll(templateDir, 0755); err != nil {
			o.log("Failed to create template directory: %v", err)
		}
	}

	// Watch Original directory if configured
	originalDir := o.cfg.Application.Storage.Original
	if originalDir != "" {
		if err := os.MkdirAll(originalDir, 0755); err != nil {
			o.log("Failed to create original directory: %v", err)
		} else {
			// Recursively watch Original directory
			err = filepath.Walk(originalDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return watcher.Add(path)
				}
				return nil
			})
			if err != nil {
				o.log("Failed to watch original directory: %v", err)
			} else {
				o.log("Watching Original directory: %s", originalDir)
			}
		}
	}

	// Set search path for this observer session
	o.db.Exec("SET search_path TO slideforge, public")

	o.log("Background observer started")

	// Initial scan - Only if auto-process is enabled
	var autoProcessVal float64
	err = o.db.QueryRow("SELECT value FROM search_settings WHERE key = 'auto_process_enabled'").Scan(&autoProcessVal)
	if err == nil {
		if autoProcessVal >= 0.5 {
			o.scanDirectory(stageDir, false)
			if originalDir != "" {
				o.scanDirectory(originalDir, false)
			}
		} else {
			o.log("Auto-process disabled at startup. Skipping initial scan.")
		}
	} else if err == sql.ErrNoRows {
		// Default to enabled if setting missing? Or disabled?
		// Current logic seems to want it disabled if set to 0.1
		// Let's assume missing = enabled for now as it was before,
		// but checking error is safer.
		o.scanDirectory(stageDir, false)
		if originalDir != "" {
			o.scanDirectory(originalDir, false)
		}
	} else {
		o.log("Error checking auto-process setting: %v. Proceeding with scan.", err)
		o.scanDirectory(stageDir, false)
		if originalDir != "" {
			o.scanDirectory(originalDir, false)
		}
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if strings.HasSuffix(strings.ToLower(event.Name), ".pptx") {
					// Check if auto-process is enabled
					var autoProcessVal float64
					o.db.QueryRow("SELECT value FROM search_settings WHERE key = 'auto_process_enabled'").Scan(&autoProcessVal)
					if autoProcessVal < 0.5 && autoProcessVal != 0 {
						o.log("Auto-process disabled. Skipping: %s", event.Name)
						continue
					}

					o.log("Detected change in: %s", event.Name)
					// Debounce/delay for file transfer to complete
					time.Sleep(2 * time.Second)
					o.ProcessFile(event.Name, false)
				} else if event.Has(fsnotify.Create) {
					// If a new directory is created, watch it too
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						watcher.Add(event.Name)
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			o.log("Watcher error: %v", err)

		case <-ctx.Done():
			return nil
		}
	}
}

func (o *Observer) scanDirectory(dir string, force bool) {
	o.log("Scanning directory recursively: %s", dir)
	var pptxFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			o.log("Error walking path %s: %v", path, err)
			return nil // continue walking
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".pptx") {
			pptxFiles = append(pptxFiles, path)
		}
		return nil
	})
	if err != nil {
		o.log("Critical error during scan of %s: %v", dir, err)
	}

	o.mu.Lock()
	o.totalQueued += len(pptxFiles)
	o.mu.Unlock()

	for _, fullPath := range pptxFiles {
		o.ProcessFile(fullPath, force)
	}
}

func (o *Observer) ProcessFile(path string, force bool) {
	filename := filepath.Base(path)

	o.mu.Lock()
	o.activeTasks++
	o.currentFile = filename
	o.startTime = time.Now()
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		o.activeTasks--
		if o.activeTasks == 0 {
			o.currentFile = ""
		}
		if o.totalQueued > 0 {
			o.totalQueued--
		}
		o.mu.Unlock()
	}()

	// 1. Prepare local SSD workspace
	localWorkBase := filepath.Join(o.cfg.Application.Storage.Local, "work")
	os.MkdirAll(localWorkBase, 0755)
	localPPTXPath := filepath.Join(localWorkBase, filename)

	// 2. Stream remote -> local + SHA256 in one pass
	o.log("Staging and hashing: %s", filename)
	checksum, err := o.streamAndHash(path, localPPTXPath)
	if err != nil {
		o.log("Failed to stage/hash %s: %v. Falling back to slow processing.", filename, err)
		// If streaming failed, we might still want to try direct processing (old way) or just fail
		return
	}
	defer os.Remove(localPPTXPath)

	// 3. Process-once & Change detection (using the checksum from local)
	if !force {
		existing, err := database.GetPPTXByOriginalPath(o.db, path)
		if err == nil && existing != nil {
			if existing.Checksum == checksum && checksum != "" {
				o.log("File already processed and unchanged: %s. Skipping.", filename)
				return
			}
			o.log("File changed: %s. Reprocessing.", filename)
		} else if checksum != "" {
			// Check for content match elsewhere (deduplication)
			existing, err = database.GetPPTXByChecksum(o.db, checksum)
			if err == nil && existing != nil {
				o.log("Content already processed at %s: %s. Skipping.", existing.OriginalFilePath, filename)
				return
			}
		}
	}

	o.log("Processing (Local-First): %s", filename)
	processPath := localPPTXPath

	// 4. Extract Tags (Fast on local SSD)
	tags, err := pptx.ExtractTags(processPath)
	if err != nil {
		o.log("Failed to extract tags from %s: %v", filename, err)
	}

	// 5. Extraction (CPU/IO Intensive - Fast on local SSD)
	// Identify relative path from source root to maintain folder structure
	srcRoot := o.cfg.Application.Storage.Stage
	if o.cfg.Application.Storage.Original != "" && strings.HasPrefix(path, o.cfg.Application.Storage.Original) {
		srcRoot = o.cfg.Application.Storage.Original
	} else if o.cfg.Application.Storage.Template != "" && strings.HasPrefix(path, o.cfg.Application.Storage.Template) {
		srcRoot = o.cfg.Application.Storage.Template
	}

	relPPTX, _ := filepath.Rel(srcRoot, path)
	relSubDir := filepath.Dir(relPPTX)
	if relSubDir == "." {
		relSubDir = ""
	}

	cleanFilename := strings.TrimSuffix(filename, filepath.Ext(filename))
	// Standardize segments to avoid encoding issues
	sanitize := func(s string) string {
		return strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
				return r
			}
			return '_'
		}, s)
	}

	cleanFilename = sanitize(cleanFilename)
	// Sanitize relSubDir segments
	var cleanRelParts []string
	for _, part := range strings.Split(relSubDir, string(os.PathSeparator)) {
		if part == "" || part == "." {
			continue
		}
		cleanRelParts = append(cleanRelParts, sanitize(part))
	}
	cleanRelDir := filepath.Join(cleanRelParts...)
	thumbSubPath := filepath.Join(cleanRelDir, cleanFilename)

	localThumbDir := filepath.Join(o.cfg.Application.Storage.Local, "thumbnails", thumbSubPath)
	os.MkdirAll(localThumbDir, 0755)

	// Create thumbnails on local SSD
	pngFiles, err := pptx.ExtractSlidesToPNG(processPath, localThumbDir)
	if err != nil {
		o.log("Failed to extract thumbnails from %s: %v", filename, err)
	}

	// Extract Slide Content (Text & Styles & Comments)
	slideDataMap, err := pptx.ExtractSlideContent(processPath)
	if err != nil {
		o.log("Failed to extract slide content from %s: %v", filename, err)
	}

	// Try to find comments for this file
	allComments := make(map[int][]pptx.Comment)
	for i, data := range slideDataMap {
		if len(data.Comments) > 0 {
			allComments[i] = data.Comments
		}
	}

	metadata := map[string]interface{}{
		"tags":         tags,
		"comments":     allComments,
		"processed_at": time.Now().Format(time.RFC3339),
	}
	metadataJSON, _ := json.Marshal(metadata)

	if len(allComments) > 0 {
		o.log("Found comments on %d slides in %s", len(allComments), filename)
	}

	// Database persist

	// Database persist
	pptxFile := &database.PPTXFile{
		Filename:         filename,
		OriginalFilePath: path,
		ThumbnailDirPath: thumbSubPath,
		Metadata:         metadataJSON,
		IsTemplate:       len(tags) > 0,
		Checksum:         checksum,
	}

	// Check for existing file by Checksum
	var fileID int
	err = o.db.QueryRow("SELECT id FROM pptx_files WHERE checksum = $1", checksum).Scan(&fileID)
	if err == nil && !force {
		o.log("File %s (checksum: %s) already exists (ID: %d). Skipping duplicate processing.", filename, checksum, fileID)
		o.finalizeFile(path, filename, fileID)
		return
	}

	// Fallback to filename/path check if checksum logic didn't hit
	var existingID int
	if fileID == 0 {
		err = o.db.QueryRow("SELECT id FROM pptx_files WHERE filename = $1 AND original_file_path = $2", filename, path).Scan(&existingID)
		if err == nil && existingID != 0 && !force {
			o.log("File %s already exists by path (ID: %d). Skipping.", filename, existingID)
			o.finalizeFile(path, filename, existingID)
			return
		}
		if err == nil {
			fileID = existingID
		}
	}

	if fileID == 0 {
		fileID, err = database.SavePPTXMetadata(o.db, pptxFile)
		if err != nil {
			o.log("Failed to save metadata to DB: %v", err)
			return
		}
	} else {
		// Update existing (e.g. metadata or checksum if it was empty)
		_, err = o.db.Exec("UPDATE pptx_files SET metadata = $1, is_template = $2, thumbnail_dir_path = $3, checksum = $4 WHERE id = $5",
			pptxFile.Metadata, pptxFile.IsTemplate, pptxFile.ThumbnailDirPath, pptxFile.Checksum, fileID)
		if err != nil {
			o.log("Failed to update metadata in DB: %v", err)
		}

		// If we are updating, we MIGHT want to reprocess slides if forced, but for now we assume simple idempotency
		// For safety, let's delete existing slides so we don't duplicate them if we continue
		_, _ = o.db.Exec("DELETE FROM collected_slides WHERE pptx_file_id = $1", fileID)
	}

	// Save slides and collect summaries
	var slideSummaries []string
	ctx := context.Background()

	for i, png := range pngFiles {
		slideNum := i + 1
		content := ""
		styleJSON := []byte("{}")
		slideSummary := ""

		slideTitle := fmt.Sprintf("Slide %d", slideNum)

		if data, ok := slideDataMap[slideNum]; ok {
			content = data.Text
			if sj, err := json.Marshal(data.Styles); err == nil {
				styleJSON = sj
			}

			// Generate slide summary & title
			var aiEnabledVal float64
			o.db.QueryRow("SELECT value FROM search_settings WHERE key = 'ai_insights_enabled'").Scan(&aiEnabledVal)
			aiEnabled := aiEnabledVal > 0.5 || (aiEnabledVal == 0 && o.cfg.AI.Enabled)

			if content != "" && aiEnabled {
				// Summary
				result, err := o.aiClient.SummarizeText(ctx, content)
				if err == nil {
					slideSummary = result.Content
					slideSummaries = append(slideSummaries, result.Content)
				} else {
					o.log("Failed to summarize slide %d of %s: %v", slideNum, filename, err)
				}
			}

			// Title generation
			if len(data.Comments) > 0 {
				// Priority 1: Comments
				var commentTexts []string
				for _, c := range data.Comments {
					commentTexts = append(commentTexts, c.Text)
				}
				fullComments := strings.Join(commentTexts, " | ")
				rawTitleResult, err := o.aiClient.ExtractTitleFromComments(ctx, fullComments)
				if err == nil && rawTitleResult.Content != "" {
					slideTitle = fmt.Sprintf("%d. %s", slideNum, rawTitleResult.Content)
				} else {
					o.log("Failed to extract title from comments for slide %d: %v", slideNum, err)
				}
			} else if content != "" && aiEnabled {
				// Priority 2: Slide Content
				rawTitleResult, err := o.aiClient.ExtractSlideTitle(ctx, content)
				if err == nil && rawTitleResult.Content != "" {
					slideTitle = fmt.Sprintf("%d. %s", slideNum, rawTitleResult.Content)
				}
			}
		}

		comments := ""
		if data, ok := slideDataMap[slideNum]; ok {
			var commentTexts []string
			for _, c := range data.Comments {
				commentTexts = append(commentTexts, c.Text)
			}
			comments = strings.Join(commentTexts, " | ")
		}

		// Sync thumbnail to remote storage
		relPNGPath := filepath.Base(png)
		remoteThumbDir := filepath.Join(o.cfg.Application.Storage.Thumbnails, thumbSubPath)
		os.MkdirAll(remoteThumbDir, 0755)
		remotePNGPath := filepath.Join(remoteThumbDir, relPNGPath)

		// Move from local to remote
		if err := moveFile(png, remotePNGPath); err != nil {
			o.log("Failed to sync thumbnail %s to remote: %v", relPNGPath, err)
		}

		err = database.SaveSlide(o.db, &database.Slide{
			PPTXFileID: fileID,
			SlideNum:   slideNum,
			PNGPath:    filepath.ToSlash(filepath.Join("thumbnails", thumbSubPath, relPNGPath)),
			Content:    content,
			StyleInfo:  styleJSON,
			AIAnalysis: []byte("{}"),
			AISummary:  slideSummary,
			Title:      slideTitle,
			Comments:   comments,
		})
		if err != nil {
			o.log("Failed to save slide %d: %v", slideNum, err)
		}
	}

	// Generate and save presentation summary & title
	var aiEnabledVal float64
	o.db.QueryRow("SELECT value FROM search_settings WHERE key = 'ai_insights_enabled'").Scan(&aiEnabledVal)
	aiEnabled := aiEnabledVal > 0.5 || (aiEnabledVal == 0 && o.cfg.AI.Enabled)

	if len(slideSummaries) > 0 && aiEnabled {
		// Summary
		fullTextForSummary := strings.Join(slideSummaries, "\n")
		overallSummaryResult, err := o.aiClient.SummarizeText(ctx, "This is a summary of all slides in a presentation. Please provide a high-level summary of the entire deck: \n"+fullTextForSummary)
		if err == nil {
			database.UpdatePPTXSummary(o.db, fileID, overallSummaryResult.Content)
		} else {
			o.log("Failed to generate overall summary for %s: %v", filename, err)
		}

		// Title
		if data, ok := slideDataMap[1]; ok && data.Text != "" {
			titleResult, err := o.aiClient.ExtractTitle(ctx, data.Text)
			if err == nil && titleResult.Content != "" {
				database.UpdatePPTXTitle(o.db, fileID, titleResult.Content)
			}
		}
	}

	// Cleanup local thumbnails if they were created
	os.RemoveAll(localThumbDir)

	o.log("Successfully processed: %s (Tags: %v)", filename, tags)

	// Move file to Template directory IF NOT from Original directory
	o.finalizeFile(path, filename, fileID)
}

func (o *Observer) finalizeFile(path, filename string, fileID int) {
	originalDir := o.cfg.Application.Storage.Original
	if originalDir != "" && strings.HasPrefix(path, originalDir) {
		o.log("Original storage mode: keeping file at %s", path)
		return
	}

	stageDir := o.cfg.Application.Storage.Stage
	relDir := ""
	if stageDir != "" && strings.HasPrefix(path, stageDir) {
		relPath, err := filepath.Rel(stageDir, path)
		if err == nil {
			relDir = filepath.Dir(relPath)
		}
	}

	destDir := filepath.Join(o.cfg.Application.Storage.Template, relDir)
	os.MkdirAll(destDir, 0755)

	newPath := filepath.Join(destDir, filename)

	// If path is already newPath, we are done
	if path == newPath {
		return
	}

	err := moveFile(path, newPath)
	if err != nil {
		o.log("Failed to move %s to template folder: %v", filename, err)
	} else {
		o.log("Moved %s to %s", filename, newPath)

		// Update database path - STORE RELATIVE CATEGORY PATH
		catPath := o.getCategoryPath(newPath)
		_, err = o.db.Exec("UPDATE pptx_files SET original_file_path = $1 WHERE id = $2", catPath, fileID)
		if err != nil {
			o.log("Failed to update file path in DB: %v", err)
		}
	}
}

// getCategoryPath returns a path relative to the most relevant storage root, prepended with category name.
func (o *Observer) getCategoryPath(path string) string {
	if o.cfg.Application.Storage.Template != "" && strings.HasPrefix(path, o.cfg.Application.Storage.Template) {
		rel, err := filepath.Rel(o.cfg.Application.Storage.Template, path)
		if err == nil {
			return filepath.ToSlash(filepath.Join("template", rel))
		}
	}
	if o.cfg.Application.Storage.Stage != "" && strings.HasPrefix(path, o.cfg.Application.Storage.Stage) {
		rel, err := filepath.Rel(o.cfg.Application.Storage.Stage, path)
		if err == nil {
			return filepath.ToSlash(filepath.Join("stage", rel))
		}
	}
	if o.cfg.Application.Storage.Original != "" && strings.HasPrefix(path, o.cfg.Application.Storage.Original) {
		rel, err := filepath.Rel(o.cfg.Application.Storage.Original, path)
		if err == nil {
			// Use the basename of the original dir as category, or just "original"
			cat := filepath.Base(o.cfg.Application.Storage.Original)
			if cat == "." || cat == "/" {
				cat = "original"
			}
			return filepath.ToSlash(filepath.Join(cat, rel))
		}
	}
	// Fallback to absolute if no root matches
	return filepath.ToSlash(path)
}

func (o *Observer) ReprocessAll() {
	o.mu.Lock()
	o.activeTasks++
	o.startTime = time.Now()
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		o.activeTasks--
		o.mu.Unlock()
	}()

	o.log("STARTING FULL REPROCESS: Resetting state...")

	// 1. Clear database
	if err := database.ClearDatabase(o.db); err != nil {
		o.log("CRITICAL: Failed to clear database during reprocess: %v", err)
		return
	}

	stageDir := o.cfg.Application.Storage.Stage
	templateDir := o.cfg.Application.Storage.Template

	// 2. Move files from Template back to Stage
	if templateDir != "" && stageDir != "" {
		files, err := os.ReadDir(templateDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".pptx") {
					oldPath := filepath.Join(templateDir, file.Name())
					newPath := filepath.Join(stageDir, file.Name())
					if err := os.Rename(oldPath, newPath); err != nil {
						o.log("Failed to move %s back to stage: %v", file.Name(), err)
					} else {
						o.log("Moved %s back to stage for reprocessing", file.Name())
					}
				}
			}
		}
	}

	// 3. Trigger scan
	o.log("Retriggering full scan...")
	o.mu.Lock()
	o.totalQueued = 0 // Reset for multi-dir scan
	o.mu.Unlock()

	o.scanDirectory(stageDir, true)
	originalDir := o.cfg.Application.Storage.Original
	if originalDir != "" {
		o.scanDirectory(originalDir, true)
	}
}

func (o *Observer) GetStatus() (bool, string, int, time.Time) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.activeTasks > 0, o.currentFile, o.totalQueued, o.startTime
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

func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}
	// Fallback for cross-device move
	if err := copyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

func (o *Observer) streamAndHash(src, dst string) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer out.Close()

	hash := sha256.New()
	mw := io.MultiWriter(out, hash)

	_, err = io.Copy(mw, in)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), out.Close()
}

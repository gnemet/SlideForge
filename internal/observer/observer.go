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

func (o *Observer) incrementTask() {
	o.mu.Lock()
	o.activeTasks++
	o.mu.Unlock()
}

func (o *Observer) decrementTask() {
	o.mu.Lock()
	o.activeTasks--
	o.mu.Unlock()
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

	err = watcher.Add(stageDir)
	if err != nil {
		return err
	}

	o.log("Background observer started, watching: %s", stageDir)

	// Initial scan
	o.scanDirectory(stageDir)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if strings.HasSuffix(strings.ToLower(event.Name), ".pptx") {
					o.log("Detected change in: %s", event.Name)

					// Debounce/delay for file transfer to complete
					time.Sleep(2 * time.Second)
					o.processFile(event.Name)
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

func (o *Observer) scanDirectory(dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		o.log("Failed to scan directory: %v", err)
		return
	}

	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".pptx") {
			fullPath := filepath.Join(dir, f.Name())
			o.processFile(fullPath)
		}
	}
}

func (o *Observer) processFile(path string) {
	o.incrementTask()
	defer o.decrementTask()

	filename := filepath.Base(path)
	o.log("Processing file: %s", filename)

	// Extract Tags
	tags, err := pptx.ExtractTags(path)
	if err != nil {
		o.log("Failed to extract tags from %s: %v", filename, err)
	}

	metadata := map[string]interface{}{
		"tags":         tags,
		"processed_at": time.Now().Format(time.RFC3339),
	}
	metadataJSON, _ := json.Marshal(metadata)

	// Thumbnails directory
	cleanFilename := strings.TrimSuffix(filename, filepath.Ext(filename))
	thumbDir := filepath.Join(o.cfg.Application.Storage.Thumbnails, cleanFilename)

	// Create thumbnails
	pngFiles, err := pptx.ExtractSlidesToPNG(path, thumbDir)
	if err != nil {
		o.log("Failed to extract thumbnails from %s: %v", filename, err)
	}

	// Extract Slide Content (Text & Styles)
	slideDataMap, err := pptx.ExtractSlideContent(path)
	if err != nil {
		o.log("Failed to extract slide content from %s: %v", filename, err)
	}

	// Calculate Checksum (SHA256)
	fileBytes, err := os.ReadFile(path)
	checksum := ""
	if err == nil {
		hash := sha256.Sum256(fileBytes)
		checksum = hex.EncodeToString(hash[:])
	} else {
		o.log("Failed to read file for checksum %s: %v", filename, err)
	}

	// Database persist
	pptxFile := &database.PPTXFile{
		Filename:         filename,
		OriginalFilePath: path,
		ThumbnailDirPath: cleanFilename,
		Metadata:         metadataJSON,
		IsTemplate:       len(tags) > 0,
		Checksum:         checksum,
	}

	// Check for existing file by Checksum
	var fileID int
	if checksum != "" {
		err = o.db.QueryRow("SELECT id FROM pptx_files WHERE checksum = $1", checksum).Scan(&fileID)
		if err == nil {
			o.log("File %s (checksum: %s) already exists (ID: %d). Skipping duplicate processing.", filename, checksum, fileID)
			o.finalizeFile(path, filename, fileID)
			return
		}
	}

	// Fallback to filename/path check if checksum logic didn't hit
	var existingID int
	err = o.db.QueryRow("SELECT id FROM pptx_files WHERE filename = $1 AND original_file_path = $2", filename, path).Scan(&existingID)

	if err == sql.ErrNoRows || existingID == 0 {
		fileID, err = database.SavePPTXMetadata(o.db, pptxFile)
		if err != nil {
			o.log("Failed to save metadata to DB: %v", err)
			return
		}
	} else if err == nil {
		fileID = existingID
		// Update existing (e.g. metadata or checksum if it was empty)
		_, err = o.db.Exec("UPDATE pptx_files SET metadata = $1, is_template = $2, thumbnail_dir_path = $3, checksum = $4 WHERE id = $5",
			pptxFile.Metadata, pptxFile.IsTemplate, pptxFile.ThumbnailDirPath, pptxFile.Checksum, fileID)
		if err != nil {
			o.log("Failed to update metadata in DB: %v", err)
		}

		// If we are updating, we MIGHT want to reprocess slides if forced, but for now we assume simple idempotency
		// For safety, let's delete existing slides so we don't duplicate them if we continue
		_, _ = o.db.Exec("DELETE FROM collected_slides WHERE pptx_file_id = $1", fileID)
	} else {
		o.log("DB error checking existing file: %v", err)
		return
	}

	// Save slides and collect summaries
	var slideSummaries []string
	ctx := context.Background()

	for i, png := range pngFiles {
		slideNum := i + 1
		relPath, err := filepath.Rel(o.cfg.Application.Storage.Thumbnails, png)
		if err != nil {
			o.log("Failed to get relative path for %s: %v", png, err)
			relPath = png // fallback
		}

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
			if content != "" {
				// Summary
				summary, err := o.aiClient.SummarizeText(ctx, content)
				if err == nil {
					slideSummary = summary
					slideSummaries = append(slideSummaries, summary)
				} else {
					o.log("Failed to summarize slide %d of %s: %v", slideNum, filename, err)
				}

				// Title
				rawTitle, err := o.aiClient.ExtractSlideTitle(ctx, content)
				if err == nil && rawTitle != "" {
					slideTitle = fmt.Sprintf("%d. %s", slideNum, rawTitle)
				}
			}
		}

		err = database.SaveSlide(o.db, &database.Slide{
			PPTXFileID: fileID,
			SlideNum:   slideNum,
			PNGPath:    "/thumbnails/" + relPath,
			Content:    content,
			StyleInfo:  styleJSON,
			AIAnalysis: []byte("{}"),
			AISummary:  slideSummary,
			Title:      slideTitle,
		})
		if err != nil {
			o.log("Failed to save slide %d: %v", slideNum, err)
		}
	}

	// Generate and save presentation summary & title
	if len(slideSummaries) > 0 {
		// Summary
		fullTextForSummary := strings.Join(slideSummaries, "\n")
		overallSummary, err := o.aiClient.SummarizeText(ctx, "This is a summary of all slides in a presentation. Please provide a high-level summary of the entire deck: \n"+fullTextForSummary)
		if err == nil {
			database.UpdatePPTXSummary(o.db, fileID, overallSummary)
		} else {
			o.log("Failed to generate overall summary for %s: %v", filename, err)
		}

		// Title
		if data, ok := slideDataMap[1]; ok && data.Text != "" {
			title, err := o.aiClient.ExtractTitle(ctx, data.Text)
			if err == nil && title != "" {
				database.UpdatePPTXTitle(o.db, fileID, title)
			}
		}
	}

	o.log("Successfully processed: %s (Tags: %v)", filename, tags)

	// Move file to Template directory
	o.finalizeFile(path, filename, fileID)
}

func (o *Observer) finalizeFile(path, filename string, fileID int) {
	if o.cfg.Application.Storage.Template == "" {
		return
	}

	newPath := filepath.Join(o.cfg.Application.Storage.Template, filename)

	// If path is already newPath, we are done
	if path == newPath {
		return
	}

	err := os.Rename(path, newPath)
	if err != nil {
		o.log("Failed to move %s to template folder: %v", filename, err)
	} else {
		o.log("Moved %s to %s", filename, newPath)

		// Update database path
		_, err := o.db.Exec("UPDATE pptx_files SET original_file_path = $1 WHERE id = $2", newPath, fileID)
		if err != nil {
			o.log("Failed to update file path in DB: %v", err)
		}
	}
}

func (o *Observer) ReprocessAll() {
	o.incrementTask()
	defer o.decrementTask()

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
	o.log("Retriggering full scan of %s", stageDir)
	o.scanDirectory(stageDir)
}

func (o *Observer) IsProcessing() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.activeTasks > 0
}

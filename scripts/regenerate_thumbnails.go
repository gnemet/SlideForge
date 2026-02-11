package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gnemet/SlideForge/internal/config"
	"github.com/gnemet/SlideForge/internal/pptx"
	_ "github.com/lib/pq"
)

func resolvePath(cfg *config.Config, relPath string) string {
	if filepath.IsAbs(relPath) {
		return relPath
	}
	if strings.HasPrefix(relPath, "stage/") {
		return filepath.Join(cfg.Application.Storage.Stage, strings.TrimPrefix(relPath, "stage/"))
	}
	if strings.HasPrefix(relPath, "original/") {
		// Note: we need to use the specific original storage path
		// In our .env, STORAGE_ORIGINAL points to the FDD folder.
		return filepath.Join(cfg.Application.Storage.Original, strings.TrimPrefix(relPath, "original/"))
	}
	if strings.HasPrefix(relPath, "template/") {
		return filepath.Join(cfg.Application.Storage.Template, strings.TrimPrefix(relPath, "template/"))
	}
	return relPath
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.Database.GetConnectStr())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, filename, original_file_path, thumbnail_dir_path FROM slideforge.pptx_files")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var filename, originalPath, thumbDirPath string
		if err := rows.Scan(&id, &filename, &originalPath, &thumbDirPath); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		absOriginalPath := resolvePath(cfg, originalPath)
		absThumbDir := filepath.Join(cfg.Application.Storage.Thumbnails, thumbDirPath)
		checkFile := filepath.Join(absThumbDir, "slide-0001.png")

		if _, err := os.Stat(checkFile); os.IsNotExist(err) {
			if _, err := os.Stat(absOriginalPath); os.IsNotExist(err) {
				log.Printf("ID %d: CANNOT REGENERATE. Original file not found at %s", id, absOriginalPath)
				continue
			}

			fmt.Printf("ID %d: Thumbnail missing for %s. Regenerating from %s...\n", id, filename, absOriginalPath)

			// Ensure thumb dir exists
			os.MkdirAll(absThumbDir, 0755)

			_, err := pptx.ExtractSlidesToPNG(absOriginalPath, absThumbDir, cfg.Application.Storage.Temp)
			if err != nil {
				log.Printf("ID %d: Failed to regenerate: %v", id, err)
			} else {
				fmt.Printf("ID %d: Successfully regenerated thumbnails.\n", id)
			}
		}
	}
}

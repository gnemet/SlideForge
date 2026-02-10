package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, s)
}

func getThumbSubPath(originalPath string) string {
	// Root paths from .env (hardcoded for repair script simplicity)
	stageRoot := "/home/gnemet/GitHub/SlideForge/mnt/bdo/stage"
	originalRoot := "/home/gnemet/GitHub/SlideForge/mnt/bdo/FDD ( pénzügyi és adóátvilágítás )"

	purePath := originalPath

	if strings.HasPrefix(originalPath, originalRoot) {
		purePath = strings.TrimPrefix(originalPath, originalRoot)
	} else if strings.HasPrefix(originalPath, stageRoot) {
		purePath = strings.TrimPrefix(originalPath, stageRoot)
	} else {
		// Fallbacks for categorized paths or other absolute paths
		if strings.HasPrefix(originalPath, "template/") {
			purePath = strings.TrimPrefix(originalPath, "template/")
		} else if strings.HasPrefix(originalPath, "stage/") {
			purePath = strings.TrimPrefix(originalPath, "stage/")
		} else if strings.HasPrefix(originalPath, "original/") {
			purePath = strings.TrimPrefix(originalPath, "original/")
		} else {
			// Handle generic BDO mount path
			parts := strings.Split(originalPath, "/mnt/bdo/")
			if len(parts) > 1 {
				purePath = parts[1]
			}
		}
	}

	purePath = strings.TrimPrefix(purePath, "/")

	parts := strings.Split(purePath, "/")
	lastIdx := len(parts) - 1
	filename := parts[lastIdx]
	cleanFilename := strings.TrimSuffix(filename, filepath.Ext(filename))
	cleanFilename = sanitize(cleanFilename)

	var cleanRelParts []string
	for i := 0; i < lastIdx; i++ {
		part := parts[i]
		if part == "" || part == "." {
			continue
		}
		cleanRelParts = append(cleanRelParts, sanitize(part))
	}

	cleanRelDir := filepath.Join(cleanRelParts...)
	return filepath.Join(cleanRelDir, cleanFilename)
}

func main() {
	godotenv.Load(".env")
	dbURL := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("PG_HOST"), os.Getenv("PG_PORT"), os.Getenv("PG_USER"), os.Getenv("PG_PASSWORD"), os.Getenv("PG_DB"))

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Use explicit schema prefix in queries to be safe
	rows, err := db.Query("SELECT id, original_file_path, filename FROM slideforge.pptx_files")
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	defer rows.Close()

	fmt.Println("Updating thumbnail paths in slideforge.pptx_files...")
	for rows.Next() {
		var id int
		var originalPath, filename string
		if err := rows.Scan(&id, &originalPath, &filename); err != nil {
			log.Fatal(err)
		}

		newThumbPath := getThumbSubPath(originalPath)
		if newThumbPath == "" {
			cleanFilename := strings.TrimSuffix(filename, filepath.Ext(filename))
			newThumbPath = sanitize(cleanFilename)
		}

		fmt.Printf("ID %d: %s -> %s\n", id, originalPath, newThumbPath)

		_, err = db.Exec("UPDATE slideforge.pptx_files SET thumbnail_dir_path = $1 WHERE id = $2", newThumbPath, id)
		if err != nil {
			log.Printf("Failed to update ID %d: %v", id, err)
		}
	}
	fmt.Println("Done.")
}

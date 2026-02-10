package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load(".env")
	dbURL := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("PG_HOST"), os.Getenv("PG_PORT"), os.Getenv("PG_USER"), os.Getenv("PG_PASSWORD"), os.Getenv("PG_DB"))

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT filename, thumbnail_dir_path FROM pptx_files LIMIT 20")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Filename | ThumbnailDirPath")
	fmt.Println("---------------------------")
	for rows.Next() {
		var filename, thumbDir string
		if err := rows.Scan(&filename, &thumbDir); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s | %s\n", filename, thumbDir)
	}
}

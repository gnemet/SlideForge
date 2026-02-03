package database

import (
	"database/sql"
	"encoding/json"
	"time"
)

type PPTXFile struct {
	ID               int             `json:"id"`
	Filename         string          `json:"filename"`
	OriginalFilePath string          `json:"original_file_path"`
	TemplateFilePath string          `json:"template_file_path"`
	ThumbnailDirPath string          `json:"thumbnail_dir_path"`
	Metadata         json.RawMessage `json:"metadata"`
	IsTemplate       bool            `json:"is_template"`
	AISummary        string          `json:"ai_summary"`
	Title            string          `json:"title"`
	Checksum         string          `json:"checksum"`
	CreatedAt        time.Time       `json:"created_at"`
}

type Slide struct {
	ID         int             `json:"id"`
	PPTXFileID int             `json:"pptx_file_id"`
	SlideNum   int             `json:"slide_number"`
	PNGPath    string          `json:"png_path"`
	Content    string          `json:"content"`
	StyleInfo  json.RawMessage `json:"style_info"`
	AIAnalysis json.RawMessage `json:"ai_analysis"`
	AISummary  string          `json:"ai_summary"`
	Title      string          `json:"title"`
	CreatedAt  time.Time       `json:"created_at"`
}

type PPTXWithSlides struct {
	PPTXFile
	Slides []Slide `json:"slides"`
}

func SavePPTXMetadata(db *sql.DB, f *PPTXFile) (int, error) {
	query := `
		INSERT INTO pptx_files (filename, original_file_path, thumbnail_dir_path, metadata, is_template, ai_summary, title, checksum)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	var id int
	err := db.QueryRow(query, f.Filename, f.OriginalFilePath, f.ThumbnailDirPath, f.Metadata, f.IsTemplate, f.AISummary, f.Title, f.Checksum).Scan(&id)
	return id, err
}

func GetPPTXByChecksum(db *sql.DB, checksum string) (*PPTXFile, error) {
	var f PPTXFile
	query := "SELECT id, filename, original_file_path, thumbnail_dir_path, is_template, metadata, ai_summary, title, checksum, created_at FROM pptx_files WHERE checksum = $1"
	err := db.QueryRow(query, checksum).Scan(&f.ID, &f.Filename, &f.OriginalFilePath, &f.ThumbnailDirPath, &f.IsTemplate, &f.Metadata, &f.AISummary, &f.Title, &f.Checksum, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func UpdatePPTXTitle(db *sql.DB, id int, title string) error {
	_, err := db.Exec("UPDATE pptx_files SET title = $1 WHERE id = $2", title, id)
	return err
}

func UpdatePPTXSummary(db *sql.DB, id int, summary string) error {
	_, err := db.Exec("UPDATE pptx_files SET ai_summary = $1 WHERE id = $2", summary, id)
	return err
}

func SaveSlide(db *sql.DB, s *Slide) error {
	query := `
		INSERT INTO collected_slides (pptx_file_id, slide_number, png_path, content, style_info, ai_analysis, ai_summary, title)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := db.Exec(query, s.PPTXFileID, s.SlideNum, s.PNGPath, s.Content, s.StyleInfo, s.AIAnalysis, s.AISummary, s.Title)
	return err
}

func GetSlidesByFile(db *sql.DB, fileID int) ([]Slide, error) {
	rows, err := db.Query("SELECT id, pptx_file_id, slide_number, png_path, content, style_info, ai_analysis, ai_summary, title, created_at FROM collected_slides WHERE pptx_file_id = $1 ORDER BY slide_number", fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slides []Slide
	for rows.Next() {
		var s Slide
		if err := rows.Scan(&s.ID, &s.PPTXFileID, &s.SlideNum, &s.PNGPath, &s.Content, &s.StyleInfo, &s.AIAnalysis, &s.AISummary, &s.Title, &s.CreatedAt); err != nil {
			return nil, err
		}
		slides = append(slides, s)
	}
	return slides, nil
}

func GetAllPPTX(db *sql.DB) ([]PPTXFile, error) {
	rows, err := db.Query("SELECT id, filename, original_file_path, thumbnail_dir_path, is_template, metadata, ai_summary, title, created_at FROM pptx_files ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []PPTXFile
	for rows.Next() {
		var f PPTXFile
		if err := rows.Scan(&f.ID, &f.Filename, &f.OriginalFilePath, &f.ThumbnailDirPath, &f.IsTemplate, &f.Metadata, &f.AISummary, &f.Title, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func GetAllPPTXWithSlides(db *sql.DB) ([]PPTXWithSlides, error) {
	files, err := GetAllPPTX(db)
	if err != nil {
		return nil, err
	}

	var result []PPTXWithSlides
	for _, f := range files {
		slides, err := GetSlidesByFile(db, f.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, PPTXWithSlides{
			PPTXFile: f,
			Slides:   slides,
		})
	}
	return result, nil
}

func GetTotalSlideCount(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM collected_slides").Scan(&count)
	return count, err
}

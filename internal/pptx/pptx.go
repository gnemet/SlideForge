package pptx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ExtractSlidesToPNG converts a PPTX file to a series of PNG images using LibreOffice and pdftoppm.
func ExtractSlidesToPNG(pptxPath, outputDir string) ([]string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %v", err)
	}

	tempPDFDir := filepath.Join("temp", "pdf")
	if err := os.MkdirAll(tempPDFDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp pdf dir: %v", err)
	}
	// We don't RemoveAll(tempPDFDir) here because it might contain other concurrent conversion's files
	// but we should probably clean up the specific PDF we create.
	// Actually, the original defer os.RemoveAll(tempPDFDir) was safe because MkdirTemp created a unique dir.
	// Since we want to use a shared temp/pdf, we should create a subfolder for this specific task.

	uniqueTaskDir, err := os.MkdirTemp(tempPDFDir, "task_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create unique task dir in temp/pdf: %v", err)
	}
	defer os.RemoveAll(uniqueTaskDir)

	tempPDFDir = uniqueTaskDir

	// Step 1: PPTX to PDF using LibreOffice
	cmd := exec.Command("libreoffice", "--headless", "--convert-to", "pdf", "--outdir", tempPDFDir, pptxPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("libreoffice conversion failed: %v (output: %s)", err, string(output))
	}

	pdfName := filepath.Base(pptxPath)
	pdfName = pdfName[:len(pdfName)-len(filepath.Ext(pdfName))] + ".pdf"
	pdfPath := filepath.Join(tempPDFDir, pdfName)

	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		// List files in tempPDFDir to see what happened
		var foundFiles []string
		if entries, err := os.ReadDir(tempPDFDir); err == nil {
			for _, entry := range entries {
				foundFiles = append(foundFiles, entry.Name())
			}
		}
		return nil, fmt.Errorf("pdf file not found after conversion: %v (expected %s, found: %v)", pdfPath, pdfName, foundFiles)
	}

	// Step 2: PDF to PNG using pdftoppm
	outputBase := filepath.Join(outputDir, "slide")
	cmd = exec.Command("pdftoppm", "-png", "-rx", "150", "-ry", "150", pdfPath, outputBase)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm conversion failed: %v", err)
	}

	// Step 3: Rename slide-N.png to slide-000N.png for better sorting
	files, err := filepath.Glob(filepath.Join(outputDir, "slide-*.png"))
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`slide-(\d+)\.png$`)
	for _, f := range files {
		matches := re.FindStringSubmatch(f)
		if len(matches) > 1 {
			num, _ := strconv.Atoi(matches[1])
			newPath := filepath.Join(outputDir, fmt.Sprintf("slide-%04d.png", num))
			os.Rename(f, newPath)
		}
	}

	// Return numerically sorted file list
	finalFiles, _ := filepath.Glob(filepath.Join(outputDir, "slide-*.png"))
	sort.Strings(finalFiles)

	return finalFiles, nil
}

// ExtractTags finds all {{tag}} patterns in the PPTX slides.
func ExtractTags(pptxPath string) ([]string, error) {
	r, err := zip.OpenReader(pptxPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	tagRegex := regexp.MustCompile(`{{(.*?)}}`)
	tagMap := make(map[string]bool)

	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}

			matches := tagRegex.FindAllStringSubmatch(string(content), -1)
			for _, match := range matches {
				if len(match) > 1 {
					tagMap[match[1]] = true
				}
			}
		}
	}

	var tags []string
	for tag := range tagMap {
		tags = append(tags, tag)
	}
	return tags, nil
}

// SlideData holds extracted text and style information for a slide.
type SlideData struct {
	SlideNumber int
	Text        string
	Styles      interface{} // Changed to interface{} to support JSONSlide structure
	Comments    []Comment   `json:"comments,omitempty"`
}

// Comment represents a PPTX comment.
type Comment struct {
	Author string    `json:"author"`
	Text   string    `json:"text"`
	Date   time.Time `json:"date,omitempty"`
}

// CommentAuthor represents a PPTX comment author.
type CommentAuthor struct {
	ID   string
	Name string
}

// Structures for rich JSON extraction
type JSONSlide struct {
	Index  int     `json:"index"`
	Shapes []Shape `json:"shapes"`
}

type Shape struct {
	Type string    `json:"type"` // title | body | other
	Runs []TextRun `json:"runs"`
}

type TextRun struct {
	Text  string `json:"text"`
	Bold  bool   `json:"bold,omitempty"`
	Size  int    `json:"size,omitempty"` // pt
	Font  string `json:"font,omitempty"`
	Color string `json:"color,omitempty"`
}

// ExtractSlideContent extracts text and rich structure info from all slides in a PPTX.
func ExtractSlideContent(pptxPath string) (map[int]SlideData, error) {
	r, err := zip.OpenReader(pptxPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	result := make(map[int]SlideData)

	// Extract authors first
	authors, _ := ExtractAuthors(pptxPath)

	for _, f := range r.File {
		// Proper check for slide files: starts with ppt/slides/slide and ends with .xml
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			// Extract index from filename, e.g., ppt/slides/slide1.xml -> 1
			baseName := filepath.Base(f.Name)
			numStr := strings.TrimSuffix(strings.TrimPrefix(baseName, "slide"), ".xml")
			slideNum, err := strconv.Atoi(numStr)
			if err != nil {
				continue
			}

			rc, err := f.Open()
			if err != nil {
				continue
			}

			jsonSlide, plainText, err := parseSlideXML(rc, slideNum)
			rc.Close()
			if err != nil {
				continue // skip on error
			}

			// Try to find comments for this slide
			comments, _ := ExtractCommentsForSlide(pptxPath, slideNum, authors)

			// Fallback/Add Notes as well
			notes, _ := ExtractNotesForSlide(pptxPath, slideNum)
			for _, n := range notes {
				comments = append(comments, Comment{Author: "Presenter Note", Text: n})
			}

			result[slideNum] = SlideData{
				SlideNumber: slideNum,
				Text:        strings.TrimSpace(plainText),
				Styles:      jsonSlide, // Store the rich structure here
				Comments:    comments,
			}
		}
	}

	return result, nil
}

// ExtractAuthors extracts comment authors from ppt/commentAuthors.xml
func ExtractAuthors(pptxPath string) (map[string]string, error) {
	r, err := zip.OpenReader(pptxPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	authors := make(map[string]string)

	for _, f := range r.File {
		if f.Name == "ppt/commentAuthors.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			dec := xml.NewDecoder(rc)
			for {
				tok, err := dec.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					return nil, err
				}

				if el, ok := tok.(xml.StartElement); ok && el.Name.Local == "cmAuthor" {
					var id, name string
					for _, a := range el.Attr {
						switch a.Name.Local {
						case "id":
							id = a.Value
						case "name":
							name = a.Value
						}
					}
					if id != "" {
						authors[id] = name
					}
				}
			}
			break
		}
	}

	return authors, nil
}

// ExtractCommentsForSlide extracts comments for a specific slide index.
// Slide index for comments usually matches slide index in ppt/slides/slide[N].xml
// Comments are in ppt/comments/comment[N].xml and linked via ppt/slides/_rels/slide[N].xml.rels
// However, a simpler heuristic is often ppt/comments/comment[N].xml mapping to slide[N].xml
func ExtractCommentsForSlide(pptxPath string, slideNum int, authors map[string]string) ([]Comment, error) {
	r, err := zip.OpenReader(pptxPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var comments []Comment

	// First, find the comment file related to this slide
	// PPTX structure: ppt/slides/_rels/slide[N].xml.rels contains target="../../comments/comment[M].xml"
	relPath := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", slideNum)
	var commentFileName string

	for _, f := range r.File {
		if f.Name == relPath {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			dec := xml.NewDecoder(rc)
			for {
				tok, err := dec.Token()
				if err == io.EOF {
					break
				}
				if el, ok := tok.(xml.StartElement); ok && el.Name.Local == "Relationship" {
					var target, rType string
					for _, a := range el.Attr {
						switch a.Name.Local {
						case "Target":
							target = a.Value
						case "Type":
							rType = a.Value
						}
					}
					if strings.HasSuffix(rType, "comments") {
						// target is like "../../comments/comment1.xml"
						commentFileName = filepath.Join("ppt", "slides", target)
						// Clean up path: ppt/slides/../../comments/comment1.xml -> ppt/comments/comment1.xml
						commentFileName = filepath.Clean(commentFileName)
						break
					}
				}
			}
			rc.Close()
			break
		}
	}

	if commentFileName == "" {
		return nil, nil
	}

	// Now parse the comment file
	for _, f := range r.File {
		if f.Name == commentFileName {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			dec := xml.NewDecoder(rc)
			for {
				tok, err := dec.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					return nil, err
				}

				if el, ok := tok.(xml.StartElement); ok && el.Name.Local == "cm" {
					var authorId, text, dateStr string
					for _, a := range el.Attr {
						switch a.Name.Local {
						case "authorId":
							authorId = a.Value
						case "dt":
							dateStr = a.Value
						}
					}

					// Find text in <text> tag inside <cm>
					// Or more simply, continue decoding until </cm>
				innerLoop:
					for {
						innerTok, err := dec.Token()
						if err == io.EOF {
							break
						}
						switch innerEl := innerTok.(type) {
						case xml.StartElement:
							if innerEl.Name.Local == "text" {
								var t string
								if err := dec.DecodeElement(&t, &innerEl); err == nil {
									text = t
								}
							}
						case xml.EndElement:
							if innerEl.Name.Local == "cm" {
								break innerLoop
							}
						}
					}

					authorName := authors[authorId]
					if authorName == "" {
						authorName = "Unknown"
					}

					var date time.Time
					if dateStr != "" {
						// PPTX date format: 2024-05-14T12:00:00.000
						date, _ = time.Parse("2006-01-02T15:04:05.000", dateStr)
					}

					comments = append(comments, Comment{
						Author: authorName,
						Text:   text,
						Date:   date,
					})
				}
			}
			break
		}
	}

	return comments, nil
}

// ExtractNotesForSlide extracts speaker notes for a slide index.
// Slide index for notes usually matches slide index in ppt/notesSlides/notesSlide[N].xml
// Note: This is a heuristic mapping.
func ExtractNotesForSlide(pptxPath string, slideNum int) ([]string, error) {
	r, err := zip.OpenReader(pptxPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var notes []string
	notesPath := fmt.Sprintf("ppt/notesSlides/notesSlide%d.xml", slideNum)

	for _, f := range r.File {
		if f.Name == notesPath {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			dec := xml.NewDecoder(rc)
			for {
				tok, err := dec.Token()
				if err == io.EOF {
					break
				}
				if el, ok := tok.(xml.StartElement); ok && el.Name.Local == "t" {
					var t string
					if err := dec.DecodeElement(&t, &el); err == nil {
						if strings.TrimSpace(t) != "" {
							notes = append(notes, strings.TrimSpace(t))
						}
					}
				}
			}
			break
		}
	}
	return notes, nil
}

func parseSlideXML(r io.Reader, index int) (*JSONSlide, string, error) {
	dec := xml.NewDecoder(r)

	slide := &JSONSlide{Index: index}
	var textBuilder strings.Builder

	var currentShape *Shape
	var currentRun *TextRun
	var placeholderType string

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}

		switch el := tok.(type) {

		case xml.StartElement:
			switch el.Name.Local {

			case "ph": // placeholder (title/body)
				for _, a := range el.Attr {
					if a.Name.Local == "type" {
						placeholderType = a.Value
					}
				}

			case "sp": // shape
				currentShape = &Shape{
					Type: normalizePlaceholder(placeholderType),
				}

			case "r": // text run
				currentRun = &TextRun{}

			case "rPr": // run formatting
				if currentRun != nil {
					for _, a := range el.Attr {
						switch a.Name.Local {
						case "b":
							currentRun.Bold = a.Value == "1"
						case "sz":
							if sz, err := strconv.Atoi(a.Value); err == nil {
								currentRun.Size = sz / 100 // 1/100 pt
							}
						}
					}
				}

			case "latin": // font family
				if currentRun != nil {
					for _, a := range el.Attr {
						if a.Name.Local == "typeface" {
							currentRun.Font = a.Value
						}
					}
				}

			case "srgbClr": // color
				if currentRun != nil {
					for _, a := range el.Attr {
						if a.Name.Local == "val" {
							currentRun.Color = "#" + a.Value
						}
					}
				}

			case "t": // actual text
				if currentRun != nil {
					var text string
					if err := dec.DecodeElement(&text, &el); err == nil {
						currentRun.Text = text
					}
				}
			}

		case xml.EndElement:
			switch el.Name.Local {

			case "r":
				if currentShape != nil &&
					currentRun != nil {
					// Append text to builder regardless of shape separation, adding space for separation
					if currentRun.Text != "" {
						currentShape.Runs = append(currentShape.Runs, *currentRun)
						textBuilder.WriteString(currentRun.Text)
						textBuilder.WriteString(" ")
					}
				}
				currentRun = nil

			case "sp":
				if currentShape != nil && len(currentShape.Runs) > 0 {
					slide.Shapes = append(slide.Shapes, *currentShape)
				}
				currentShape = nil
				placeholderType = ""
			}
		}
	}

	return slide, textBuilder.String(), nil
}

func normalizePlaceholder(ph string) string {
	switch ph {
	case "title", "ctrTitle":
		return "title"
	case "body":
		return "body"
	default:
		return "other"
	}
}

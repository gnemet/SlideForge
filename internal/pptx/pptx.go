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
)

// ExtractSlidesToPNG converts a PPTX file to a series of PNG images using LibreOffice and pdftoppm.
func ExtractSlidesToPNG(pptxPath, outputDir string) ([]string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %v", err)
	}

	tempPDFDir := filepath.Join(os.TempDir(), "slideforge_pdf")
	os.MkdirAll(tempPDFDir, 0755)
	defer os.RemoveAll(tempPDFDir)

	// Step 1: PPTX to PDF using LibreOffice
	cmd := exec.Command("libreoffice", "--headless", "--convert-to", "pdf", "--outdir", tempPDFDir, pptxPath)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("libreoffice conversion failed: %v", err)
	}

	pdfName := filepath.Base(pptxPath)
	pdfName = pdfName[:len(pdfName)-len(filepath.Ext(pdfName))] + ".pdf"
	pdfPath := filepath.Join(tempPDFDir, pdfName)

	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("pdf file not found after conversion: %v", pdfPath)
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

			result[slideNum] = SlideData{
				SlideNumber: slideNum,
				Text:        strings.TrimSpace(plainText),
				Styles:      jsonSlide, // Store the rich structure here
			}
		}
	}

	return result, nil
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

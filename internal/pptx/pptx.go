package pptx

import (
	"archive/zip"
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
		if filepath.HasPrefix(f.Name, "ppt/slides/slide") && filepath.Ext(f.Name) == ".xml" {
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
	Styles      map[string]interface{}
}

// ExtractSlideContent extracts text and style info from all slides in a PPTX.
func ExtractSlideContent(pptxPath string) (map[int]SlideData, error) {
	r, err := zip.OpenReader(pptxPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	textRegex := regexp.MustCompile(`<a:t>(.*?)</a:t>`)
	slideNumRegex := regexp.MustCompile(`ppt/slides/slide(\d+)\.xml`)

	result := make(map[int]SlideData)

	for _, f := range r.File {
		matches := slideNumRegex.FindStringSubmatch(f.Name)
		if len(matches) > 1 && filepath.Ext(f.Name) == ".xml" {
			slideNum, _ := strconv.Atoi(matches[1])

			rc, err := f.Open()
			if err != nil {
				continue
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}

			// Extract all text
			var slideText strings.Builder
			textMatches := textRegex.FindAllStringSubmatch(string(content), -1)
			for _, tm := range textMatches {
				if len(tm) > 1 {
					slideText.WriteString(tm[1])
					slideText.WriteString(" ")
				}
			}

			// Basic style info (heuristic: count specific tags or attributes)
			styleInfo := make(map[string]interface{})
			if strings.Contains(string(content), "sz=\"") {
				styleInfo["has_custom_font_size"] = true
			}
			if strings.Contains(string(content), "<a:schemeClr") {
				styleInfo["uses_theme_colors"] = true
			}

			result[slideNum] = SlideData{
				SlideNumber: slideNum,
				Text:        strings.TrimSpace(slideText.String()),
				Styles:      styleInfo,
			}
		}
	}

	return result, nil
}

package pptx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	// Step 2: PDF to PNG using pdftoppm
	// -png: output PNG
	// -rx 150 -ry 150: resolution
	outputBase := filepath.Join(outputDir, "slide")
	cmd = exec.Command("pdftoppm", "-png", "-rx", "150", "-ry", "150", pdfPath, outputBase)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftoppm conversion failed: %v", err)
	}

	// Find generated files
	files, err := filepath.Glob(filepath.Join(outputDir, "slide-*.png"))
	if err != nil {
		return nil, err
	}

	return files, nil
}

// Placeholder for PPTX XML manipulation
func CreateTemplate(pptxPath, templatePath string) error {
    // Logic to unzip, find text in slides, and mark as template
    // For now, just copy as placeholder
    data, err := os.ReadFile(pptxPath)
    if err != nil {
        return err
    }
    return os.WriteFile(templatePath, data, 0644)
}

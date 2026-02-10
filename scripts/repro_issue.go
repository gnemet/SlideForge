package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	pptxName := "Test_&_Symbol.pptx"
	err := os.WriteFile(pptxName, []byte("fake pptx content"), 0644)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer os.Remove(pptxName)

	tempDir, _ := os.MkdirTemp("", "slideforge_test_*")
	defer os.RemoveAll(tempDir)

	fmt.Printf("Converting %s to PDF in %s...\n", pptxName, tempDir)

	cmd := exec.Command("libreoffice", "--headless", "--convert-to", "pdf", "--outdir", tempDir, pptxName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("LibreOffice failed: %v\nOutput: %s\n", err, string(output))
		return
	}

	files, _ := os.ReadDir(tempDir)
	fmt.Println("Files in temp dir:")
	for _, f := range files {
		fmt.Printf(" - %s\n", f.Name())
	}

	expectedPDF := "Test_&_Symbol.pdf"
	pdfPath := filepath.Join(tempDir, expectedPDF)
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		fmt.Printf("❌ FAILED: Expected %s not found!\n", expectedPDF)
	} else {
		fmt.Printf("✅ SUCCESS: Found %s\n", expectedPDF)
	}
}

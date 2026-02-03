package mcp

import (
	"embed"
	"fmt"
)

// CatalogFS contains the pre-compiled JSON catalogs
// This is generated at build time using Go's embed feature.
//
//go:embed catalog/*.json
var CatalogFS embed.FS

// CatalogProvider handles extraction of metadata from embedded files
type CatalogProvider struct{}

func NewCatalogProvider() *CatalogProvider {
	return &CatalogProvider{}
}

// GetCatalogMetadata reads a JSON catalog from the embedded filesystem
func (p *CatalogProvider) GetCatalogMetadata(resourceCode string) (string, error) {
	fileName := fmt.Sprintf("catalog/%s.json", resourceCode)
	content, err := CatalogFS.ReadFile(fileName)
	if err != nil {
		return "", fmt.Errorf("could not read embedded catalog file %s: %w", fileName, err)
	}

	return string(content), nil
}

// GetAllCatalogs returns a list of available resources
func (p *CatalogProvider) GetAllCatalogs() ([]string, error) {
	entries, err := CatalogFS.ReadDir("catalog")
	if err != nil {
		return nil, err
	}
	var codes []string
	for _, e := range entries {
		if !e.IsDir() {
			// basics: remove .json extension
			name := e.Name()
			if len(name) > 5 {
				codes = append(codes, name[:len(name)-5])
			}
		}
	}
	return codes, nil
}

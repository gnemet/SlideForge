package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/gnemet/SlideForge/internal/i18n"
	"github.com/gnemet/SlideForge/internal/mcp"
	"github.com/gnemet/datagrid"
)

// ResourcePageHandler renders the management page for a specific resource
// e.g. /resource?code=pptx_files
func handleResourcePage(w http.ResponseWriter, r *http.Request) {
	lang := i18n.GetLang(r)
	resourceCode := r.URL.Query().Get("code")
	if resourceCode == "" {
		http.Error(w, "Missing resource code", http.StatusBadRequest)
		return
	}

	// 1. Fetch Metadata from Catalog
	provider := mcp.NewCatalogProvider()
	meta, err := provider.GetCatalogMetadata(resourceCode)
	if err != nil {
		log.Printf("Resource not found: %s", resourceCode)
		http.NotFound(w, r)
		return
	}

	// 2. Initialize Datagrid Config
	dgHandler, err := datagrid.NewHandlerFromData(sqlDB, []byte(meta), lang)
	if err != nil {
		log.Printf("Error configuring datagrid: %v", err)
		http.Error(w, "Configuration Error", http.StatusInternalServerError)
		return
	}

	// 3. Prepare Data
	icon := dgHandler.Catalog.Icon
	if icon == "" {
		icon = "ph-database"
	}

	// Ensure icon has ph- prefix if missing (legacy support)
	if len(icon) > 3 && icon[:3] != "ph-" && icon[:3] != "fa-" {
		// Assume it might be just "presentation", prepend ph-
		// But let's trust the catalog first
	}

	data := getBaseData(r, dgHandler.Catalog.Title, "tables")

	// Add Datagrid specific data
	data["ResourceCode"] = resourceCode
	data["Datagrid"] = dgHandler.Config
	data["ListEndpoint"] = "/resource/list?code=" + resourceCode
	data["UIColumns"] = dgHandler.Columns
	data["Icon"] = icon
	data["Title"] = dgHandler.Catalog.Title
	data["Limit"] = dgHandler.Config.Defaults.PageSize
	if data["Limit"] == 0 {
		data["Limit"] = 10
	}
	data["Offset"] = 0

	renderTemplate(w, "resource.html", data)
}

// ResourceListHandler - HTMX Partial for the grid
func handleResourceList(w http.ResponseWriter, r *http.Request) {
	lang := i18n.GetLang(r)
	resourceCode := r.URL.Query().Get("code")
	if resourceCode == "" {
		http.Error(w, "Missing resource code", http.StatusBadRequest)
		return
	}

	// 1. Load Catalog
	provider := mcp.NewCatalogProvider()
	meta, err := provider.GetCatalogMetadata(resourceCode)
	if err != nil {
		log.Printf("Resource not found: %s", resourceCode)
		http.NotFound(w, r)
		return
	}

	// 2. Initialize Handler
	dgHandler, err := datagrid.NewHandlerFromData(sqlDB, []byte(meta), lang)
	if err != nil {
		log.Printf("Error creating handler: %v", err)
		http.Error(w, "Config error", http.StatusInternalServerError)
		return
	}

	// 3. Fetch Data
	params := dgHandler.ParseParams(r)
	result, err := dgHandler.FetchData(params)
	if err != nil {
		log.Printf("Error fetching data: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Render Partial (from Datagrid Embedded Assets)
	funcMap := template.FuncMap{
		"T": func(key string) string { return i18n.T(lang, key) },
	}
	// Merge global funcs
	for k, v := range datagrid.TemplateFuncs() {
		funcMap[k] = v
	}

	tmpl, err := template.New("table.html").Funcs(funcMap).ParseFS(datagrid.UIAssets, "ui/templates/partials/datagrid/table.html")
	if err != nil {
		log.Printf("Error parsing embedded template: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "datagrid_table", result); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

// Helper: getCatalogItems returns metadata for all registered catalogs
func getCatalogItems() []map[string]string {
	provider := mcp.NewCatalogProvider()
	codes, err := provider.GetAllCatalogs()
	if err != nil {
		return nil
	}

	var items []map[string]string
	for _, code := range codes {
		meta, _ := provider.GetCatalogMetadata(code)
		if meta == "" {
			continue
		}
		var manifest datagrid.Catalog
		if err := json.Unmarshal([]byte(meta), &manifest); err == nil {
			items = append(items, map[string]string{
				"Code":  code,
				"Title": manifest.Title,
				"Icon":  manifest.Icon,
			})
		}
	}
	return items
}

// MetaPageHandler renders the list of all tables/catalogs
func handleMetaPage(w http.ResponseWriter, r *http.Request) {
	items := getCatalogItems()
	data := getBaseData(r, "Table Management", "tables")
	data["Catalogs"] = items

	renderTemplate(w, "meta.html", data)
}

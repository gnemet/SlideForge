# Datagrid Integration

## Overview
SlideForge integrates the `gnemet/datagrid` library to provide a powerful, sortable, and filterable table view of the PowerPoint library.

## Configuration
The datagrid is initialized in `cmd/server/main.go`:

```go
dgHandler := datagrid.NewHandler(sqlDB, "pptx_files", []datagrid.UIColumn{
    {Field: "filename", Label: "Name", Visible: true, Sortable: true},
    {Field: "created_at", Label: "Uploaded", Visible: true, Sortable: true},
    {Field: "is_template", Label: "Template", Visible: true, Type: "boolean"},
}, datagrid.DatagridConfig{})
```

## UI Integration
*   **Sidebar**: A "Table" item has been added to the main navigation (`/templates`).
*   **Routing**: The handler is served at `/templates` via `AuthMiddleware(dgHandler.ServeHTTP)`.
*   **Library**: Uses the local replacement of `github.com/gnemet/datagrid` to ensure compatibility with the "Library-First" architecture.

---
description: pptx selector and generator
---

# UI
- ** left tree ** show pptx filename/title , children slides, show title or slide id
- ** right tree ** show slides with compound name:  pptx name / slide name
- ** minimal button ** use icons instead of text ( global rule too )
- ** danger action menu ** use a dropdown menu for destruction actions (remove, clean, delete). **NO individual action buttons on rows.**

# UX 
- ** context menu **
## Slides
- ** click ** select (exclusive)
- ** ctrl/cmd-click ** multi-select (toggle)
- ** shift-click ** show slide metadata
- ** alt-click ** show slide image preview
- ** alt-shift-click ** show slide JSON data
- ** drag and drop ** copy to collection (preserve order)
- ** double click ** copy to collection
- ** filter ** filter the slides, show count of slide and pptx

## Collection
- ** click ** select
- ** ctrl-click ** multi-select
- ** double click ** delete from collection
- ** drag and drop ** reorder within collection, or drag out to delete
- ** right click ** show context menu for all actions (Preview, Meta, JSON, Delete)
- ** NO ACTION BUTTONS ** items must be clean rows without per-item buttons.
- ** menu ** delete selected, clean all 

## Viewer
- ** resizable & draggable ** all modals (Meta, JSON, Preview) must be resizable and draggable for advanced workspaces.
- ** json syntax highlighting ** use color-coded highlighting for JSON data views (Blue: keys, Green: strings, Orange: numbers).
- ** auto-reset ** modals must reset to centered/default size on close to maintain workspace cleanliness.

# Technical Rules
- ** i18n syntax ** use backticks for all translation keys `{{T .Lang \`key\` }}` to prevent attribute quote-clashing and runtime panics.
- ** visibility logic ** use utility classes `.hidden` and `.visible` instead of complex logic in inline `style` attributes to ensure IDE lint cleanliness.
- ** dynamic counters ** use `data-suffix` pattern for localizing interactive JS-driven counters (Slides/PPTX counts).
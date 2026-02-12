---
trigger: always_on
---

# UI/UX Rules

## Design Philosophy
- **Modern & Elegant**: Enterprise-grade, clean, and minimal dashboard.
- **Card-Based Layout**: Use generous whitespace and subtle shadows.
- **Flat UI**: Low contrast, no gradients, no glossy effects, and no clutter.
- **Theme Support**: Native Dark and Light modes on all pages.
- **Language Support**: Language selection modes on all pages. Use the language resource instead of hardcode text 

## Typography
- **System Font Stack**: Avoid custom font loads for speed.
- **Hierarchy**: Semibold headings and section titles; regular body text.

## Icons
- **Font Awesome Solid**: Use `fas` classes only.
- **Monochrome**: Avoid colorful or custom SVG icons.

## Layout & UX
- **Responsive 12-Column Grid**: Content-first design.
- **Minimal Animations**: Subtle hover states only.
- **Desktop-First**: Optimized for large screen productivity.

## Interactions
- **HTMX**: Dynamic updates via AJAX for tables and cards.
- **jQuery**: Lightweight DOM manipulation and event handling.
- **Loop-Selector**: Use a single loop-selector button instead of a dropdown/chooser for lists with few items (e.g., Language, Theme).
- **Frameworks**: Avoid full JS frameworks (React/Vue).
- **Tooling**: Built-in formatters and validators for SQL and JSON.

## Components
- **Buttons**: ALWAYS use Font Awesome icons and `title` attributes instead of button text if an icon exists. Avoid redundant text on action buttons.
- **Text, Textbox**: Use icons instead of external labels. Labels should be placed inside the input field as `placeholder` text (floating icons preferred).
- **Reusability**: Standardize cards, tables, and modals across pages.
- **Footer**: Sticky footer with app name, version, DB status, and build date.

## Clean Code
- **ABSOLUTE NO HARDCODING**: **NEVER** hardcode UI strings, labels, icons, or status colors in Go, HTMX (templates), or JS.
- **Metadata-Driven UI**: All UI elements (icons, multi-language labels, LOVs, formatters) MUST be provided by the backend via metadata (e.g., JSON catalogs or API response headers).
- **HTMX Templates**: NEVER hardcode HTML strings in Go code (handlers). ALWAYS use external `.html` templates and render them with data.
- **No Inline Styles/Scripts**: All styling must be in `.css` files and all logic in `.js` files. 
- **Separation of Concerns**: Keep HTML structure, CSS styling, and JS behavior strictly separated.


## Context Menu Usage

The agent MUST prefer using a custom context menu (right-click, secondary click, or long-press) whenever a UI element supports multiple actions.

Rules:
- Group all secondary or contextual actions inside a context menu.
- Expose only the primary action as a direct click when applicable.
- Avoid showing multiple inline buttons that cause UI clutter.
- Override the default browser context menu when meaningful custom actions exist.
- Use consistent context menu behavior across the application.

When to use a context menu:
- List items (e.g. Edit, Delete, Duplicate)
- Table rows (e.g. View, Export, Archive)
- Visual or editor elements (e.g. Transform, Layer, Properties)

Exceptions:
- When an element has only one clear action.
- When using a context menu would significantly reduce discoverability.

Accessibility requirements:
- All context menu actions MUST be keyboard-accessible.
- Provide an alternative trigger (keyboard shortcut or menu button).
- Use appropriate ARIA roles for menus and menu items.

Note:
- On touch devices, a long-press MUST trigger the same context menu behavior.
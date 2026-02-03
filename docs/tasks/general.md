# SlideForge Task Registry

## 🛠 Active Workflows
- **[PPTX Generator](/pptx-generator)**: Standardized tree views, minimal iconography, and premium UX interactions (Context Menus, Shift-Click, Double-Click).
- **[UI/UX Refinement](/ui-ux-frontend)**: Font Awesome rollback, dynamic layout synchronization, and branding cleanliness.

## ⚙️ Environment Configuration

### Authentication
Manage the security gateway of your workshop via `.env` or `config.yaml`:
- `AUTH_TYPE=none`: Disables security checks.
- `AUTH_TYPE=jwt`: Enables JWT-based session security (Default).
- `AUTH_ENABLED=false`: Global toggle to disable authentication middleware.

### Database
- **Schema**: `slideforge`
- **Options**: `search_path=slideforge,public`

### Storage Hierarchy
- `/uploads`: Incoming PPTX raw files.
- `/thumbnails`: Atomic PNG slide extraction.
- `/templates`: Design-system PPTX files.

## 📝 Roadmap & Fixes
- [x] Standardize on Font Awesome `fas` icons.
- [x] Resolve "Double Icon" branding issues.
- [x] Decouple Table Management from hardcoded list to dynamic metadata.
- [x] Implement Minimal Button (Icon-Only) policy across all hubs.
- [x] Fix Alt+Shift+Click for JSON metadata in Generator.
- [x] Implement Danger Action menu for clean/remove operations.

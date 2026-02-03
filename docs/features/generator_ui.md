# Generator UI & UX Features

## Overview
The Generator interface allows users to compose new presentations by selecting slides from the library. It features an advanced drag-and-drop system and context-aware interactions.

## Key Features

### 1. Global Multi-Select Drag & Drop
*   **Selection**: Users can select multiple slides across **different** presentations using `Ctrl + Click`.
*   **Global Drag**: Dragging *any* selected slide (from any presentation container) will gather **ALL** currently selected slides from the entire library and drop them into the collection target together.
*   **Order Preservation**: Slides are added in the order they appear in the source tree.

### 2. Context Menu
*   **Right-Click**: Opens a custom context menu on any slide.
*   **Actions**:
    *   **Preview**: Shows a high-res modal of the slide.
    *   **Metadata**: Displays AI-extracted summary and raw content.
    *   **Add to Collection**: Adds the slide to the target deck.
    *   **Remove**: Removes the slide (if in the target collection).

### 3. Quick Actions (Click Behavior)
*   **Click**: Selects the slide (Exclusive selection).
*   **Ctrl + Click**: Toggles selection (Multi-select).
*   **Alt + Click**: Opens the **Slide Preview** modal instantly.
*   **Double Click**: (Standard text selection behavior is preserved).

# System Process Flow

This diagram illustrates the end-to-end lifecycle of a presentation file within the SlideForge ecosystem.

```mermaid
graph TD
    A[User Uploads PPTX] -->|Watch Folder| B(Observer Service)
    B --> C{File Type?}
    C -->|Invalid| D[Log & Skip]
    C -->|PPTX| E[Move to Stage]
    
    E --> F[Unzip & Parse XML]
    F --> G[Convert Slides to PNG]
    G --> H[Store in /thumbnails]
    H --> I[Extract Raw Text]
    I --> J[Calculate Checksum]
    
    J --> K{Exists in DB?}
    K -->|Yes| L[Move to Template/Archive]
    K -->|No| M[Save Metadata to PG]
    
    M --> N[Trigger AI Analysis]
    N --> O[Searchable Data Ready]
    
    subgraph "Visual Pipeline"
    G
    H
    end
    
    subgraph "Data Pipeline"
    F
    I
    J
    end
    
    subgraph "Persistence Tier"
    M
    O
    end
```

## Data Transformation Pipeline

1.  **Binary Content**: The raw `.pptx` file is unzipped and analyzed.
2.  **Visual Layer**: Each slide is rendered to image for UI previews.
3.  **Semantic Layer**: AI models (Gemini/OpenAI) generate summaries and tags.
4.  **Vector Layer**: Content is embedded for similarity search.

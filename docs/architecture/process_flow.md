# System Process Flow

This diagram illustrates the end-to-end lifecycle of a presentation file within the SlideForge ecosystem.

```mermaid
graph TD
    A[User Uploads PPTX] -->|Watch Folder| B(Observer Service)
    B --> C{File Type?}
    C -->|Invalid| D[Log & Skip]
    C -->|PPTX| E[Move to Stage]
    
    E --> F[Convert to PNGs]
    F --> G[Extract Raw Text]
    G --> H[Calculate Checksum]
    
    H --> I{Exists in DB?}
    I -->|Yes| J[Move to Archive]
    I -->|No| K[Save Metadata to PG]
    
    K --> L[Trigger AI Analysis]
    L --> M[Searchable Data Ready]
    
    subgraph "Processing Tier"
    F
    G
    H
    end
    
    subgraph "Persistence Tier"
    K
    M
    end
```

## Data Transformation Pipeline

1.  **Binary Content**: The raw `.pptx` file is unzipped and analyzed.
2.  **Visual Layer**: Each slide is rendered to image for UI previews.
3.  **Semantic Layer**: AI models (Gemini/OpenAI) generate summaries and tags.
4.  **Vector Layer**: Content is embedded for similarity search.

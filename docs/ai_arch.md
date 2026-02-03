# 🏗️ AI Cognitive Architecture: The Digital Blacksmith

Our ecosystem utilizes a sophisticated multi-turn AI reasoning engine powered by **Google Gemini**. By treating the AI as a "Digital Blacksmith," we forge high-precision outputs (SQL or PPTX logic) through a structured, iterative process.

## 🧠 Core Methodology

### 1. Model Context Protocol (MCP)
The foundation of our AI's intelligence is the **MCP**. Instead of sending raw prompts, we first initialize a multi-turn conversation with the full system metadata:
*   **DWH Catalog**: Consolidated JSON schemas describing every table and column.
*   **Domain Rules**: Architectural constraints (e.g., PostgreSQL focus, DWH conventions).
*   **Instructional Persona**: Setting the AI as a Senior DWH Expert.

### 2. Multi-Turn Context Persistence
Unlike simple "one-shot" API calls, we use Gemini's **Chat Session** feature:
1.  **Turn 1 (The Initialization)**: We send the "Big MCP" (Schema Catalog) for analysis.
2.  **Turn 2 (The Acknowledgment)**: Gemini confirms receipt and structures the metadata internally.
3.  **Turn 3 (The Request)**: The user sends a natural language question (e.g., *"Show me all worklogs"*).
4.  **The Result**: Because the AI already "understands" the database from Turn 1, the generated SQL is accurate, schema-aware, and optimized.

## 🛠️ Specialized Tooling

### AI-SQL Laboratory (JIRAMNTR)
*   **Dynamic Metadata**: Filters out UI noise (datagrid) to focus 100% on DDL metadata.
*   **Few-Shot Examples**: Includes verified SQL patterns to guide the AI's edge-case handling.
*   **Audit Logging**: Every request is logged for performance and accuracy monitoring.

### AI Chat Tester (SlideForge)
*   **Dynamic Model Selection**: Switches between `gemini-1.5-flash`, `pro`, and `latest` based on complexity.
*   **Real-Time Indicators**: Utilizes a "Live Status" pulse (Animated CSS) and real-time balance tracking.
*   **Diagnostic Verifiers**: Integrated scripts ensure sub-500ms connectivity.

## 💰 Financial Transparency & Health
We've integrated the **Google Cloud Billing API** directly into our UI badges:
*   **Live Token Monitoring**: Estimated balance and budget are displayed in real-time.
*   **Service Account Isolation**: Credentials are sequestered in `opt/` and protected by `.gitignore`.
*   **System Pulse**: A glassmorphism-style badge provides instant visual confirmation of AI availability.

---
*Document Version: 1.2.0 (Feb 2026)*
*Powered by Antigravity Engine*

# Cyber Security Agent API

The agent exposes a REST API via HTTP (default port: 8080).

## Base URL
`http://localhost:8080`

## Endpoints

### 1. Super Agent Orchestration (Main)
Synthesizes reports from all Research Agents and optionally analyzes a GNN Figure using VLM.

-   **URL**: `/api/super/chat`
-   **Method**: `POST`
-   **Body**:
    ```json
    {
      "query": "Analyze the threat trends for the energy sector.",
      "image": "<base64_string_optional>"
    }
    ```
-   **Response**:
    ```json
    {
      "final_report": "Comprehensive synthesized report...",
      "agent_reports": {
        "attacker_feasibility": "...",
        "sector_geo_context": "..."
      },
      "vlm_analysis": "Analysis of the provided figure..."
    }
    ```

### 2. Specific Research Agent Chat
Interact with a single specialized agent.

-   **URL**: `/api/agent/{agent_id}/chat`
-   **Method**: `POST`
-   **Body**:
    ```json
    {
      "query": "What are the latest CVEs?"
    }
    ```
-   **Response**:
    ```json
    {
      "agent": "Attacker Feasibility",
      "response": "Based on Mandiant M-Trends..."
    }
    ```
-   **Agent IDs**: `attacker_feasibility`, `defensive_readiness`, `exploit_kinetics`, `sector_geo_context`, `risk_cost_impact`.

### 3. VLM Analysis (Vision Only)
Directly use the VLM to analyze an image.

-   **URL**: `/api/vlm/analyze`
-   **Method**: `POST`
-   **Body**:
    ```json
    {
      "image": "<base64_string>",
      "prompt": "Describe this network topology."
    }
    ```
-   **Response**:
    ```json
    {
      "response": "This image shows a star topology..."
    }
    ```

### 4. Ingest Documents
Ingest PDFs from a directory into a specific collection (Knowledge Base).

-   **URL**: `/api/ingest`
-   **Method**: `POST`
-   **Body**:
    ```json
    {
      "dir": "/app/doc/My_KB_Folder",
      "collection_name": "My_KB_Name"
    }
    ```

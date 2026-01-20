# Cyber Security Knowledge Base (KB) Overview

This document outlines the architecture and data sources of the Cyber Security RAG System's knowledge base. The system uses a **Dual-Database Strategy** to combine the reasoning power of natural language context with the factual accuracy of structured data.

## 1. Storage Strategy: VDB vs. RDB

| Data Source | Content Type | Primary Storage | Usage in System |
| :--- | :--- | :--- | :--- |
| **MITRE ATT&CK** | Attack Techniques & Tactics | **Vector DB (Qdrant)** | Reasoning on attack methods and detection. |
| **CISA KEV** | Vulnerability Metadata & Action | **Hybrid (VDB + RDB)** | Contextual descriptions + Exploitation status. |
| **NVD (NIST)** | CVE Technical Summaries | **Hybrid (VDB + RDB)** | Technical depth + Severity (CVSS) & Timeline. |
| **AlienVault OTX** | Threat Pulses & IOCs | **Hybrid (VDB + RDB)** | Threat background + IP/Domain indicator lookup. |
| **EPSS (FIRST)** | Exploit Probability Metrics | **RDBS (SQLite)** | Quantitative risk analysis (Kinetic). |
| **PDF Reports** | Industry/Expert Analysis | **Vector DB (Qdrant)** | Deep-dive reports (DBIR, Mandiant, etc.). |

---

## 2. Ingested Datasets (`cyber_intel_full`)

The label `cyber_intel_full` in `config.yaml` refers to the primary collection in the **Vector Database**. It aggregates the "Natural Language" part of all technical sources to provide the LLM with rich contextual information.

- **MITRE ATT&CK**: Ingested into a dedicated `mitre_attack` collection for high-fidelity technique mapping.
- **Full Intel**: The `cyber_intel_full` collection includes:
  - CISA Known Exploited Vulnerabilities (KEV) descriptions.
  - NVD CVE descriptions.
  - AlienVault OTX Pulse summaries.
  - Thousands of pages from industry reports (PDFs) processed with recursive text extraction.

---

## 3. Relational Database (RDBS) & Agent Tools

While the Vector DB provides *context*, the **SQLite RDBS** (`data/agent.db`) provides *facts*. LLM Agents access this structured data through specialized **ADK Tools**.

### A. CVE Details Tool (`check_cve_details`)
- **Source Tables**: `epss`, `cisa_kev`, `nvd_meta`
- **Condition**: Look up by `cve_id` (e.g., `CVE-2023-38831`).
- **Processing**:
  1. Retrieves the **EPSS Score** and **Percentile** to measure real-time exploitation probability.
  2. Checks if the CVE is in the **CISA KEV** catalog (confirmed exploited in the wild).
  3. Fetches the **CVSS Base Score** from NVD for severity validation.
- **LLM Context**: Injected as a structured JSON object, allowing the agent to say: *"This CVE has an EPSS score of 0.95, indicating it is highly likely to be exploited."*

### B. Indicator Search Tool (`search_indicators`)
- **Source Table**: `otx_ioc`
- **Condition**: Look up by `indicator` (IP address or Domain).
- **Processing**:
  1. Searches the OTX Indicators of Compromise table.
  2. Returns the associated **Pulse ID**, **IOC Type**, and **Creation Date**.
- **LLM Context**: Allows agents to verify if a specific infrastructure element is known to be malicious based on AlienVault's crowd-sourced intelligence.

---

## 4. Why Dual-Database?

1. **Precision**: Vector search can sometimes hallucinate or be too broad. Structured SQL queries for metrics like EPSS are 100% accurate.
2. **Speed**: Querying a CVE by ID in SQLite is significantly faster than embedding a query and searching 10k+ vectors.
3. **Logic**: LLM agents can use the quantitative data from the RDBS to perform "calculations" (e.g., *if score > 7.0 then HIGH risk*) that are difficult for raw RAG alone.

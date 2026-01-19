# Agent Architecture & Flow

This document details the multi-agent architecture of the Cyber Security RAG System.

## Agent Descriptions

The system consists of **5 Specialized Research Agents** and **1 Super Agent**.

| Agent Name | ID | Role & Description | Knowledge Base(s) |
| :--- | :--- | :--- | :--- |
| **Attacker Feasibility** | `attacker_feasibility` | **Offensive Researcher**: Analyzes the technical feasibility of attacks, considering complexity and available exploits. | `Mandiant-M-Trends`, `IBM-X-FORCE` |
| **Defensive Readiness** | `defensive_readiness` | **Defensive Specialist**: Evaluates the organization's preparedness, security policies, and infrastructure logs. | `CMU_SEI_CERT`, `DFIR` |
| **Exploit Kinetics** | `exploit_kinetics` | **Threat Intel Analyst**: Tracks the speed and trends of how exploits are adopted in the wild. | `IBM-X-FORCE`, `DFIR` |
| **Sector/Geo Context** | `sector_geo_context` | **Geopolitical Analyst**: Assesses threats based on industry sector and geographical location. | `DBIR`, `Mandiant-M-Trends` |
| **Risk & Cost Impact** | `risk_cost_impact` | **Risk Assessor**: Estimates potential financial and operational impacts of successful attacks. | `DBIR`, `DFIR` |
| **Super Agent** | `super_agent` | **Synthesizer**: Orchestrates the process, collecting reports from all research agents and GNN predictions to generate a final comprehensive report. | *N/A (Uses Agent Reports)* |

## Agent Flow Diagram

The following Mermaid diagram illustrates the flow from a user query to the final synthesized report.

```mermaid
graph TD
    User([User / API]) -->|Query| SA[Super Agent]
    
    subgraph "Orchestration & Data Gathering"
        SA -->|Delegates Task| RA1[Attacker Feasibility]
        SA -->|Delegates Task| RA2[Defensive Readiness]
        SA -->|Delegates Task| RA3[Exploit Kinetics]
        SA -->|Delegates Task| RA4[Sector/Geo Context]
        SA -->|Delegates Task| RA5[Risk & Cost Impact]
        
        SA -->|Feeds Data| GNN[GNN Model Prediction]
    end
    
    subgraph "RAG Process (Per Agent)"
        RA1 <-->|Query & Retrieve| KB1[("Mandiant-M-Trends<br/>IBM-X-FORCE")]
        RA2 <-->|Query & Retrieve| KB2[("CMU_SEI_CERT<br/>DFIR")]
        RA3 <-->|Query & Retrieve| KB3[("IBM-X-FORCE<br/>DFIR")]
        RA4 <-->|Query & Retrieve| KB4[("DBIR<br/>Mandiant-M-Trends")]
        RA5 <-->|Query & Retrieve| KB5[("DBIR<br/>DFIR")]
    end
    
    subgraph "Synthesis"
        RA1 -->|Report| Aggregator
        RA2 -->|Report| Aggregator
        RA3 -->|Report| Aggregator
        RA4 -->|Report| Aggregator
        RA5 -->|Report| Aggregator
        GNN -->|Figure/Analysis| Aggregator
        
        Aggregator[Context Aggregation] -->|Full Context| SA_Final[Super Agent Synthesis]
    end
    
    SA_Final -->|Final Threat Analysis| User
```

## Setup Note
Each agent is configured in `config.yaml` with specific instructions and mapped knowledge bases. The `doc/` directory structure must match these knowledge base names for correct ingestion.

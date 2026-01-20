# Additional Cyber Security Datasets

This document describes the additional datasets collected for threat intelligence and analysis. These datasets provide structured information about techniques, vulnerabilities, and risk scores.

## Dataset Availability Summary

| Dataset | Availability | Format | Auth Required | Status |
| :--- | :--- | :--- | :--- | :--- |
| **MITRE ATT&CK** | Yes | JSON (STIX) | No | ✅ Downloaded |
| **CISA KEV** | Yes | JSON | No | ✅ Downloaded |
| **EPSS** | Yes | CSV | No | ✅ Downloaded |
| **NVD (NIST)** | Yes | API / JSON | Recommended | ℹ️ API Access Preferred |
| **AlienVault OTX** | Yes | API | Yes (API Key) | 🔑 Key Required |
| **CVSS** | Yes | Embedded / API | No | ℹ️ Included in NVD |

---

## 1. MITRE ATT&CK
- **Description**: A globally-accessible knowledge base of adversary tactics and techniques based on real-world observations.
- **File**: `data/datasets/enterprise-attack.json`
- **Format**: STIX 2.1 JSON
- **Usage**: Used to map threat actor behaviors to specific techniques and sub-techniques.
- **Update Command**:
  ```bash
  curl -L -o data/datasets/enterprise-attack.json https://raw.githubusercontent.com/mitre/cti/master/enterprise-attack/enterprise-attack.json
  ```

## 2. CISA KEV (Known Exploited Vulnerabilities)
- **Description**: A catalog of vulnerabilities that have been actively exploited in the wild.
- **File**: `data/datasets/cisa_kev.json`
- **Format**: JSON
- **Usage**: Prioritize patching based on vulnerabilities that are known to be used by attackers.
- **Update Command**:
  ```bash
  curl -L -o data/datasets/cisa_kev.json https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json
  ```

## 3. EPSS (Exploit Prediction Scoring System)
- **Description**: An open, data-driven framework for estimating the probability that a software vulnerability will be exploited.
- **File**: `data/datasets/epss_scores.csv`
- **Format**: CSV
- **Usage**: Provides a probability score (0.0 to 1.0) for CVEs.
- **Update Command**:
  ```bash
  curl -L -o data/datasets/epss_scores.csv.gz https://epss.empiricalsecurity.com/epss_scores-current.csv.gz
  gunzip -f data/datasets/epss_scores.csv.gz
  ```

## 4. NVD (National Vulnerability Database)
- **Description**: The U.S. government repository of standards based vulnerability management data.
- **Recommendation**: Use the NVD API 2.0 for the most up-to-date information. 
- **API Endpoint**: `https://services.nvd.nist.gov/rest/json/cves/2.0`
- **Usage**: Retrieve CVSS scores, CWE mappings, and CPE information.
- **Auth**: An API key is highly recommended to avoid strict rate limiting.

## 5. AlienVault OTX (Open Threat Exchange)
- **Description**: A community-powered threat intelligence platform.
- **Integration**: Requires an OTX API Key.
- **API Documentation**: [OTX API](https://otx.alienvault.com/api/)
- **Usage**: Retrieve indicators of compromise (IoCs) like IP addresses, domains, and file hashes.

---

## How to use these datasets in Python (Example)

```python
import json
import pandas as pd

# Load CISA KEV
with open('data/datasets/cisa_kev.json', 'r') as f:
    kev_data = json.load(f)
    print(f"Total KEV vulnerabilities: {len(kev_data['vulnerabilities'])}")

# Load EPSS Scores
epss_df = pd.read_csv('data/datasets/epss_scores.csv', skiprows=1)
print(epss_df.head())
```

---

## Sample Parsing Script
A sample Go script to parse MITRE ATT&CK is available at `scripts/parse_mitre.go`.
Run it with:
```bash
go run scripts/parse_mitre.go
```

---

## CISA KEV Parsing Script
A sample Go script to parse CISA KEV is available at `scripts/parse_cisa_kev.go`.
Run it with:
```bash
go run scripts/parse_cisa_kev.go
```

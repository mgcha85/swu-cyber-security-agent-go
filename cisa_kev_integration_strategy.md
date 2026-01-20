# CISA KEV (Known Exploited Vulnerabilities) Integration Strategy

`cisa_kev.json` 데이터는 실제 전 세계적으로 **악용 중인 것으로 확인된** 취약점 리스트입니다. 이 프로젝트에서 이를 활용할 수 있는 4가지 전략을 제안합니다.

## 1. 취약점 우선순위 도구 (Risk Scoring Tool)
CVE 번호를 기반으로 위협의 심각도를 즉시 판별하는 데 사용합니다.

-   **방식**: AI 에이전트가 특정 CVE(예: CVE-2024-xxxx)를 발견했을 때, CISA KEV 리스트에 포함되어 있는지 즉석 조회합니다.
-   **활용**: 
    -   에이전트 판단 로직: "이 취약점은 CPSS 점수가 낮더라도 CISA KEV에 포함되어 있으므로 **긴급(Urgent)** 패치 대상으로 분류함."
-   **이점**: 이론적인 위험도가 아닌 **실제 공격 사례**를 기반으로 한 실전적인 대응 우선순위를 제공합니다.

## 2. 랜섬웨어 분석 강화 (Ransomware Intelligence)
일부 KEV 항목은 랜섬웨어 그룹에 의해 사용된 이력(`knownRansomwareCampaignUse`)을 포함합니다.

-   **방식**: `knownRansomwareCampaignUse` 필드가 "Known"인 취약점을 별도로 마킹합니다.
-   **활용**: "이 취약점은 현재 랜섬웨어 공격에 활발히 사용 중이므로 데이터 유출 및 암호화 위협이 매우 높음"과 같은 구체적인 경고를 보고서에 포함합니다.

## 3. RAG 컨텍스트 주입 (Contextual Retrieval)
MITRE ATT&CK과 마찬가지로 Qdrant에 저장하여 검색 결과로 제공합니다.

-   **방식**: `cveID`, `vulnerabilityName`, `shortDescription`을 벡터화하여 저장합니다.
-   **효과**: 사용자가 "최근 Microsoft 제품에서 가장 위험한 취약점이 뭐야?"라고 물으면, CISA가 공식적으로 관리하는 '실제 악용 사례'를 기반으로 답변할 수 있습니다.

## 4. 대응 가이드 자동 생성 (Remediation Automation)
CISA KEV는 각 취약점에 대해 정부가 권고하는 `requiredAction`과 `dueDate`를 포함합니다.

-   **활용**: 시스템이 분석 보고서를 생성할 때, 하단에 CISA의 공식 조치 사항(`requiredAction`)을 그대로 인용하여 신뢰도 높은 행동 지침을 제공합니다.

---

## Go 데이터 구조 정의

```go
type KevVulnerability struct {
    CveID                      string   `json:"cveID"`
    VendorProject              string   `json:"vendorProject"`
    Product                    string   `json:"product"`
    VulnerabilityName          string   `json:"vulnerabilityName"`
    ShortDescription           string   `json:"shortDescription"`
    RequiredAction             string   `json:"requiredAction"`
    DueDate                    string   `json:"dueDate"`
    KnownRansomwareCampaignUse string   `json:"knownRansomwareCampaignUse"`
    Notes                      string   `json:"notes"`
}

type KevCatalog struct {
    Vulnerabilities []KevVulnerability `json:"vulnerabilities"`
}
```

## 제안하는 연동 워크플로우
1.  **Fast Lookup**: 서버 시작 시 메모리(Map)에 `CVE-ID -> KEV Object`를 캐싱합니다.
2.  **Report Annotation**: 에이전트가 보고서를 작성할 때 CVE가 언급되면 자동으로 `[CISA KEV] - 이 취약점은 실제 악용 사례가 보고됨`이라는 태그와 함께 상세 설명을 추가합니다.
3.  **Cross-Mapping**: MITRE ATT&CK의 기법과 KEV의 CVE를 상호 연결하여 "공격자가 사용하는 T1078 기법이 실제 CVE-2024-xxxx를 통해 구현됨"과 같은 상세 위협 체인을 구축합니다.

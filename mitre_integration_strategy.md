# MITRE ATT&CK Integration Strategy

`enterprise-attack.json` 파일은 STIX 2.1 표준을 따르는 매우 방대한 데이터셋입니다. 이 프로젝트에서 이를 효과적으로 활용하기 위한 3가지 주요 전략을 제안합니다.

## 1. 전용 RAG 컬렉션 구축 (Semantic Search)
가장 보편적이고 강력한 방법입니다.

-   **방식**: Qdrant에 `mitre_attack`이라는 별도의 컬렉션을 생성하고, JSON 내의 `attack-pattern` (Technique) 오브젝트들을 인덱싱합니다.
-   **인덱싱 대상**:
    -   `name`: 기법 이름 (예: "Phishing")
    -   `description`: 상세 설명
    -   **Metadata**: `external_references`에서 기술 ID(Txxxx)를 추출하여 페이로드에 저장합니다.
-   **이점**: 에이전트가 "Adversary가 클라우드 환경에서 권한 상승을 위해 어떤 기법을 사용하는가?"와 같은 질문을 던졌을 때, MITRE의 공식 정의를 검색 결과로 제공할 수 있습니다.

## 2. 정적 식별자 매핑 (Lookup Tool)
보고서의 신뢰도를 높이기 위한 "공식 사전"으로 활용합니다.

-   **방식**: 에이전트가 조사 과정에서 특정 기술 ID(예: T1059)를 발견하면, 해당 JSON에서 즉시 `external_id`를 기반으로 기법명, 상세 설명, 대응 방안(`course-of-action`)을 찾아와 보고서에 주석(Annotate)을 답니다.
-   **이점**: 생성형 AI가 가끔 범하는 할루시네이션(잘못된 기술 ID 매칭)을 방지하고 공식 사이트 링크를 자동으로 제공할 수 있습니다.

## 3. 에이전트 페르소나 강화 (Specialization)
특정 에이전트에게 MITRE 지식을 주입합니다.

-   **대상**: `RA1: Attacker Feasibility Expert`
-   **방식**: `config.yaml`의 `knowledge_bases`에 `mitre_attack` 컬렉션을 추가하여, 공격자 관점의 분석 시 MITRE 데이터를 최우선적으로 참고하게 합니다.

---

## Ingestion 예시 코드 (Conceptual)

Go에서 이 JSON을 파싱하여 Qdrant에 넣는 구조는 다음과 같습니다.

```go
type StixObject struct {
    Type        string `json:"type"`
    Name        string `json:"name,omitempty"`
    Description string `json:"description,omitempty"`
    ExternalIds []struct {
        ID  string `json:"external_id"`
        URL string `json:"url"`
    } `json:"external_references,omitempty"`
}

type StixBundle struct {
    Objects []StixObject `json:"objects"`
}

// Ingestion Logic:
// 1. JSON 파일 로드
// 2. Objects 루프 돌며 Type == "attack-pattern" 인 것만 필터링
// 3. Name + Description으로 Embedding 생성
// 4. Qdrant mitre_attack 컬렉션에 Upsert
```

## 제안하는 다음 단계
1.  **컬렉션 생성**: Qdrant에 `mitre_attack` 컬렉션을 새로 만듭니다.
2.  **데이터 가공**: 전체 JSON(43MB) 중 `attack-pattern`, `malware`, `intrusion-set` 등 핵심 데이터만 추출하여 인덱싱하는 전용 스크립트를 작성합니다.
3.  **에이전트 연결**: `config.yaml`을 수정하여 Super Agent나 특정 전문가 에이전트가 이 컬렉션을 검색 도구로 사용하게 합니다.

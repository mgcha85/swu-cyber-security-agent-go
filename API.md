# 사이버 보안 에이전트 API

이 에이전트는 HTTP를 통해 REST API를 제공합니다 (기본 포트: 8080).

## 기본 URL (Base URL)
`http://localhost:8080`

## 엔드포인트 (Endpoints)

### 1. 슈퍼 에이전트 오케스트레이션 (메인)
모든 리서치 에이전트의 보고서를 종합하고, 선택적으로 VLM을 사용하여 GNN Figure(이미지)를 분석하여 최종 답변을 생성합니다.

-   **URL**: `/api/super/chat`
-   **Method**: `POST`
-   **Body**:
    ```json
    {
      "query": "에너지 섹터의 최신 위협 동향을 분석해줘.",
      "image": "<base64_string_optional>"
    }
    ```
    *   `query`: 사용자의 질문
    *   `image`: (선택 사항) 분석할 이미지의 Base64 인코딩 문자열 (GNN 결과 그래프 등)

-   **Response**:
    ```json
    {
      "final_report": "종합된 최종 위협 분석 보고서...",
      "agent_reports": {
        "attacker_feasibility": "공격 가능성 분석 결과...",
        "sector_geo_context": "섹터/지리적 맥락 분석 결과..."
      },
      "vlm_analysis": "제공된 이미지에 대한 분석 내용..."
    }
    ```

### 2. 개별 리서치 에이전트 채팅
특정 전문 에이전트와 단독으로 상호작용하여 테스트하거나 질의합니다.

-   **URL**: `/api/agent/{agent_id}/chat`
-   **Method**: `POST`
-   **Body**:
    ```json
    {
      "query": "최근 발견된 주요 CVE는 무엇인가?"
    }
    ```
-   **Response**:
    ```json
    {
      "agent": "Attacker Feasibility",
      "response": "Mandiant M-Trends 지식 베이스에 따르면..."
    }
    ```
-   **사용 가능한 Agent ID**:
    *   `attacker_feasibility`: 공격 실현 가능성 분석
    *   `defensive_readiness`: 방어 준비 태세 평가
    *   `exploit_kinetics`: 익스플로잇 확산 속도/추세
    *   `sector_geo_context`: 산업/지리적 위협 맥락
    *   `risk_cost_impact`: 잠재적 피해 비용 및 영향

### 3. VLM 분석 (비전 전용)
VLM 모델(Qwen3-VL 등)을 직접 호출하여 이미지를 분석합니다.

-   **URL**: `/api/vlm/analyze`
-   **Method**: `POST`
-   **Body**:
    ```json
    {
      "image": "<base64_string>",
      "prompt": "이 네트워크 토폴로지 다이어그램을 설명해줘."
    }
    ```
-   **Response**:
    ```json
    {
      "response": "이 이미지는 스타형 토폴로지를 보여주며..."
    }
    ```

### 4. 문서 수집 (Ingest)
특정 디렉토리에 있는 PDF 문서들을 지정된 컬렉션(지식 베이스)으로 임베딩하여 저장합니다.

-   **URL**: `/api/ingest`
-   **Method**: `POST`
-   **Body**:
    ```json
    {
      "dir": "/app/doc/My_KB_Folder",
      "collection_name": "My_KB_Name"
    }
    ```
    *   `dir`: PDF 파일들이 위치한 디렉토리 경로 (컨테이너 내부 경로 권장)
    *   `collection_name`: Qdrant에 생성될 컬렉션 이름 (`config.yaml`의 Knowledge Base 이름과 일치해야 함)

### 5. 메타데이터 조회
시스템의 에이전트 및 지식 베이스 정보를 조회합니다.

#### 5.1 에이전트 목록 조회
-   **URL**: `/api/agents`
-   **Method**: `GET`
-   **Response**:
    ```json
    [
      { "id": "attacker_feasibility", "name": "Attacker Feasibility" },
      { "id": "defensive_readiness", "name": "Defensive Readiness" }
    ]
    ```

#### 5.2 지식 베이스(컬렉션) 목록 조회
현재 벡터 DB(Qdrant)에 생성되어 있는 지식 베이스 목록을 반환합니다.

-   **URL**: `/api/kb`
-   **Method**: `GET`
-   **Response**:
    ```json
    {
      "knowledge_bases": [ "Mandiant-M-Trends", "IBM-X-FORCE", "DBIR" ]
    }
    ```

#### 5.3 에이전트-지식 베이스 매핑 정보 조회
각 에이전트가 어떤 지식 베이스를 사용하는지 포함한 상세 설정을 반환합니다.

-   **URL**: `/api/agents/map`
-   **Method**: `GET`
-   **Response**:
    ```json
    [
      {
        "name": "Attacker Feasibility",
        "id": "attacker_feasibility",
        "description": "...",
        "instruction": "...",
        "knowledge_bases": ["Mandiant-M-Trends", "IBM-X-FORCE"]
      }
    ]
    ```

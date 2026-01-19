# 사이버 보안 RAG 에이전트 시스템 (Go + ADK)

**Go**와 **Google Agent Development Kit (ADK)**로 구축된 사이버 보안 위협 분석 시스템입니다. RAG (검색 증강 생성)를 활용하여 여러 전문 에이전트가 PDF 보고서에서 취약점 컨텍스트를 분석할 수 있도록 지원합니다.

## 시스템 개요

이 시스템은 다음과 같이 구성됩니다:
-   **5개의 리서치 에이전트**: 각 에이전트는 특정 페르소나(예: 공격 실행 가능성, 위험 영향 등)와 전용 지식 베이스(Knowledge Base)를 가집니다.
-   **슈퍼 에이전트**: 모든 리서치 에이전트의 조사 결과를 종합하고 GNN 모델의 예측 결과와 비교 분석합니다.
-   **RAG 엔진**: **Qdrant** (Vector DB) 및 **Ollama** (임베딩)를 기반으로 동작합니다.

## 기술 스택

| 구성 요소 | 기술 | 설명 |
| :--- | :--- | :--- |
| **언어** | Go (Golang) 1.24+ | 메인 애플리케이션 로직 |
| **프레임워크** | Google ADK | 에이전트 오케스트레이션 |
| **LLM Model** | DeepSeek-V3 (OpenAI Compatible) | 로컬 LLM 추론 |
| **Embedding Model** | Qwen3-Embedding:4b (Configurable) | 임베딩 추론 |
| **Vision Model (VLM)** | Qwen3-VL:Latest | 이미지/멀티모달 분석 |
| **Vector Database** | Qdrant | PDF 임베딩 저장소 |
| **인프라** | Podman Compose | 컨테이너 관리 |

## 소스 트리 구조

```text
swu-cyber-security-agent-go/
├── cmd/
│   └── server/          # 메인 엔트리 포인트 (CLI & HTTP 서버)
├── internal/
│   ├── agent/           # 리서치 에이전트 & 슈퍼 에이전트 로직 (ADK)
│   ├── rag/             # RAG 로직 (데이터 수집, 임베딩, 검색)
│   ├── vector/          # Qdrant 클라이언트 래퍼
│   └── model/           # LLM 모델 어댑터 (Mock 포함)
├── config.yaml          # 에이전트 및 KB 설정
├── docker-compose.yml   # 인프라 정의 (Qdrant)
├── README.md            # 이 파일
├── API.md               # API 명세서
└── SETUP.md             # IDE 설정 가이드
```

## 시작하기

### 필수 요구사항
-   Go 1.24 이상
-   Podman (docker-compose shim 포함)

### 설치 방법
1.  **인프라 실행**
    ```bash
    podman-compose up -d
    ```

2.  **빌드**
    ```bash
    go build -o agent-server ./cmd/server
    ```

## 사용법

### CLI 모드

**PDF 데이터 수집 (Ingest):**
```bash
./agent-server --ingest ./your-docs-folder
```

**대화 (Chat):**
```bash
./agent-server --chat "CVE-2025-1234의 영향에 대해 분석해줘"
```

### 서버 모드
API 엔드포인트를 노출하기 위해 HTTP 서버를 시작합니다:
```bash
./agent-server --server
```
*서버는 8080 포트에서 수신 대기합니다.*
## 6. 관측 및 문서화 (Observability & Docs)

### Swagger UI (API 문서)
API 명세를 시각적으로 확인하고 테스트할 수 있습니다.
-   URL: `http://localhost:8081`

### Prometheus (메트릭 모니터링)
LLM 응답 속도 등의 성능 데이터를 조회할 수 있습니다.
-   URL: `http://localhost:9090`
-   주요 메트릭: `llm_request_duration_seconds`

## 7. 문의
문제 발생 시 Issue를 등록해주세요.
### 컨테이너 환경에서 실행 (Container)

호스트에 설치된 Ollama와 PDF 문서가 있는 `./doc` 폴더를 연동하여 컨테이너로 실행할 수 있습니다.

1.  **준비 사항**:
    -   프로젝트 루트에 `doc` 폴더 생성 및 PDF 파일 위치.
    -   호스트에서 `ollama serve`가 실행 중인지 확인 (기본 포트 11434).

2.  **실행**:
    ```bash
    podman-compose up -d --build
    ```
    *`agent` 서비스가 빌드되고 시작되며, 호스트의 `./doc` 폴더가 컨테이너의 `/app/doc`으로 마운트됩니다.*

3.  **수집 및 사용**:
    -   API를 통해 수집 명령을 내릴 때, 컨테이너 내부 경로인 `/app/doc`을 사용해야 합니다 (혹은 기본값 설정).
    -   예:
        ```bash
        curl -X POST http://localhost:8080/api/ingest -d '{"dir": "/app/doc"}'
        ```

See [API.md](API.md) for API usage details.

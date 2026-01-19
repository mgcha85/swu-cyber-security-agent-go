# API 레퍼런스

사이버 보안 에이전트 시스템은 서버 모드(`--server`)로 실행될 때 RESTful API를 제공합니다.

## 기본 URL (Base URL)
`http://localhost:8080`

## 엔드포인트

### 1. 에이전트와 대화 (Chat with Agent)
리서치 에이전트에게 질의를 보냅니다.

-   **URL**: `/api/chat`
-   **Method**: `POST`
-   **Content-Type**: `application/json`

#### 요청 본문 (Request Body)
```json
{
  "query": "질문 내용"
}
```

#### 응답 (Response)
```json
{
  "response": "지식 베이스를 기반으로 한 에이전트의 답변..."
}
```

#### 예시 (curl)
```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "query": "CVE-2025-9999 취약점을 이용한 공격의 기술적 실현 가능성을 분석해줘"
  }'
```

---

### 2. 문서 수집 (Ingest Documents)
지정된 로컬 디렉토리의 모든 PDF 파일을 벡터 데이터베이스(Vector DB)로 수집(Ingest)합니다.

-   **URL**: `/api/ingest`
-   **Method**: `POST`
-   **Content-Type**: `application/json`

#### 요청 본문 (Request Body)
```json
{
  "dir": "/absolute/path/to/docs"
}
```

#### 응답 (Response)
```json
{
  "status": "success",
  "message": "Ingestion complete"
}
```

#### 예시 (curl)
```bash
curl -X POST http://localhost:8080/api/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "dir": "/home/user/docs/cve_reports"
  }'
```

## 🛡️ GNN 모델 예측 데이터 명세

GNN(Graph Neural Network) 모델을 통해 생성되는 보안 위협 예측 결과물 및 관련 데이터 구조에 대한 정의입니다.

### 1. 트렌드 및 Gap 데이터 (`.csv`)

GNN 모델이 예측한 각 솔루션별 연도별 수치 데이터입니다. **Gap**은 트렌드 선상의 Y축 값 차이를 의미하며, 위협의 증감 추세를 나타냅니다.

* **형식:** CSV
* **구조:** `Solution, [Year1], [Year2], [Year3], ...`
* **데이터 예시:**
```csv
Solution,2025,2026,2027
Session_management,0.0002660056308438167,0.0003429859762036358,0.0004612908778653946

```



---

### 2. 파일 구성 및 명명 규칙 (File Pairing)

각 사이버 위협(Cyber Threat)에 대해 시각적 결과와 수치 결과가 한 쌍(Pair)으로 구성됩니다.

* **파일명 규칙:** `[위협_명칭].[확장자]`
* **구성 요소:**
* **PNG:** 위협 예측 트렌드를 시각화한 그래프 이미지
* **CSV:** 그래프의 기반이 되는 상세 테이블 데이터


* **예시:**
* `Account_Hijacking.png`
* `Account_Hijacking.csv`



---

### 3. 그래프 구조 정의 데이터 (`graph_new.csv`)

GNN 모델이 학습 및 예측 시 사용하는 위협(Threat)과 솔루션(Solution) 간의 연결 관계(Graph Edge)를 정의한 파일입니다.

* **데이터 구조:**
* **1st Column:** 사이버 위협 명칭 (Threat)
* **Subsequent Columns:** 해당 위협과 연결된 대응 솔루션들 (Solutions)


* **특징:** 각 행(Row)마다 연결된 솔루션의 개수가 다르므로 가변 컬럼 구조를 가짐.
* **구조 예시:**
| 위협 (Threat) | 솔루션 1 | 솔루션 2 | 솔루션 3 | ... |
| :--- | :--- | :--- | :--- | :--- |
| Account_Hijacking | Session_management | Multi_Factor_Auth | - | - |
| DDoS_Attack | Rate_Limiting | Traffic_Filtering | Load_Balancing | WAF |

---

### 4. Super Agent 활용 시나리오

1. **데이터 로드:** Super Agent는 특정 위협(예: `Account_Hijacking`) 분석 시, 해당 이름의 `.csv`와 `graph_new.csv`를 읽어들임.
2. **트렌드 파악:** CSV의 연도별 수치(Gap)를 통해 GNN이 예측하는 36개월간의 위협 증감 방향을 확인.
3. **에이전트 비교:** 5개의 독립 연구 에이전트가 Knowledge Base를 통해 도출한 의견과 GNN의 수치적 트렌드가 일치하는지 대조하여 최종 리포트 생성.

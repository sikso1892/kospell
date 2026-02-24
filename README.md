# kospell

>  비상업적 용도로만 사용할 수 있습니다.

[(주) 나라인포테크 맞춤법 검사기](https://nara-speller.co.kr/speller/) in golang

## 설치

```
go get github.com/Alfex4936/kospell
```

## 기본 사용법

### Go 라이브러리로 사용

```go
ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
defer cancel()

result, err := kospell.Check(ctx, "너는나와 kafka 머고나서")
```

### CLI 도구로 사용

```bash
# stdin에서 읽기
echo "너는나와 kafka 머고나서" | kospell-cli

# 파일에서 읽기
kospell-cli -f text.txt

# 타임아웃 설정 (기본: 8초)
kospell-cli -f text.txt -t 10s
```

## 사용자 딕셔너리 (User Dictionary)

고유명사, 복합어, 특수 용어 등 API가 오류로 지적하는 단어를 보호하려면 사용자 딕셔너리를 사용할 수 있습니다.

### 딕셔너리 파일 형식

JSON 파일에 보호할 단어들을 등록합니다:

```json
{
  "words": [
    "목제솜틀기",
    "KoNLPy",
    "FastAPI",
    "우아한형제들"
  ]
}
```

### 라이브러리 사용 (Go)

```go
// 딕셔너리 로드
dict, err := kospell.LoadDict("user_dict.json")
if err != nil {
    log.Fatal(err)
}

ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
defer cancel()

// 딕셔너리를 적용하여 검사
result, err := kospell.CheckWithDict(ctx, "목제솜틀기는 유명한 제품입니다", dict)

// 딕셔너리에 있는 단어는 Corrections에서 제외됨
// errorCount와 corrections 모두 자동 업데이트
```

또는 직접 딕셔너리를 생성:

```go
dict := kospell.NewDict("목제솜틀기", "KoNLPy", "FastAPI")
result, err := kospell.CheckWithDict(ctx, text, dict)
```

### CLI 도구 사용

```bash
# -d 플래그로 딕셔너리 파일 지정
kospell-cli -f text.txt -d user_dict.json

# 여러 옵션 함께 사용
kospell-cli -f text.txt -d user_dict.json -t 10s
```

### 동작 원리

1. 원본 텍스트를 API에 그대로 전송
2. API 응답의 각 `Correction.Origin` (API가 지적한 단어)을 딕셔너리와 비교
3. 딕셔너리에 있는 단어의 Correction 제거
4. `ErrorCount` 자동 감소
5. 모든 Correction이 제거된 Chunk는 결과에서 제외

## 출력 형식

### 샘플 응답

```json
{
  "original": "너는나와 kafka 머고나서\r\n",
  "charCount": 18,
  "chunkCount": 1,
  "corrections": [
    {
      "idx": 0,
      "input": "너는나와 kafka 머고나서\r\n",
      "items": [
        {
          "start": 0,
          "end": 4,
          "origin": "너는나와",
          "suggest": [
            "너는 나와"
          ],
          "help": "관형사형 어미 뒤에 오는 말은 띄어 씁니다.\n\n(예) 데뷔할예정(×) -> 데뷔할 예정(○)\n잘시간(×) -> 잘 시간(○)\n좋은사람(×) -> 좋은 사람(○)\n한가한때(×) -> 한가한 때(○)\n이런식으로(×) -> 이런 식으로(○)\n그런점(×) -> 그런 점(○)"
        },
        {
          "start": 5,
          "end": 10,
          "origin": "kafka",
          "suggest": [
            "Kafka",
            "KAFA",
            "kaka",
            "KFKA"
          ],
          "help": "영어 단어를 잘못 표기하셨습니다.\n\n고유명사의 첫 글자나 문장을 시작할 때는 대문자로 써야 하고, 다른 글자나 일반 명사는 소문자로 적습니다. 첫 글자만 따서 만든 단어일 때는 모두 대문자로 씁니다.\n\n(예) Seoul\nappletiME -> time\n\n로마자를 입력할 때도 한글처럼 알파벳을 잘못 입력하여 오류를 범할 수 있으므로 주의해야 합니다.\n\n(1) 알파벳의 순서를 바꿔 쓰거나 다른 글자를 입력한 오류\nadn -> and\ntheone -> throne\n\n(2) 필요한 글자가 빠지거나 불필요한 글자를 더 입력한 오류\ncomitment -> commitment\nwellcome -> welcome"
        },
        {
          "start": 11,
          "end": 15,
          "origin": "머고나서",
          "suggest": [
            "머고 나서"
          ],
          "help": "띄어쓰기 오류입니다. 대치어를 참고하여 띄어 쓰도록 합니다."
        }
      ]
    }
  ],
  "errorCount": 3
}
```

### 응답 필드 설명

| 필드 | 설명 |
|------|------|
| `original` | 원본 입력 텍스트 |
| `charCount` | 텍스트의 UTF-8 글자 수 |
| `chunkCount` | 처리된 청크 개수 (≤300 어절 단위) |
| `corrections` | 오류 목록 (빈 배열이면 오류 없음) |
| `errorCount` | 총 오류 개수 |

#### Correction 필드

| 필드 | 설명 |
|------|------|
| `start` | 오류 시작 위치 (rune 기준) |
| `end` | 오류 끝 위치 (rune 기준) |
| `origin` | 잘못된 원본 단어 |
| `suggest` | 대체 제안 목록 |
| `help` | 오류 설명 (선택사항) |

## REST API 서버

kospell을 HTTP 서버로 실행하여 다른 애플리케이션에서 사용할 수 있습니다.

### 서버 실행

```bash
# 기본 포트 8080에서 실행
kospell-server

# 커스텀 포트에서 실행
kospell-server -p 3000

# 로컬 모드 (hunspell) — <lang>.aff/.dic 파일이 있는 디렉터리 지정
# 예) /path/to/hunspell/ko.aff, /path/to/hunspell/ko.dic
kospell-server -mode local -dict /path/to/hunspell -lang ko
```

### API 엔드포인트

#### POST /v1/check-spell

맞춤법 검사 요청

**요청 예시:**
```bash
curl -X POST http://localhost:8080/v1/check-spell \
  -H "Content-Type: application/json" \
  -d '{
    "text": "너는나와 kafka 머고나서",
    "words": ["kafka"]
  }'
```

**요청 필드:**

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `text` | string | O | 검사할 텍스트 |
| `words` | string[] | X | 오류에서 제외할 단어 목록 (인라인) |
| `dict` | object | X | 사용자 딕셔너리 `{"words":[...]}` |
| `dict_path` | string | X | (deprecated) 사용자 딕셔너리 JSON 파일 경로 (서버 로컬) |
| `timeout` | int | X | 타임아웃 (초, 기본값: 8) |

**응답:**
```json
{
  "original": "너는나와 kafka 머고나서",
  "charCount": 18,
  "chunkCount": 1,
  "corrections": [
    {
      "idx": 0,
      "input": "너는나와 kafka 머고나서",
      "items": [
        {
          "start": 0,
          "end": 4,
          "origin": "너는나와",
          "suggest": ["너는 나와"],
          "help": "관형사형 어미 뒤에 오는 말은 띄어 씁니다."
        }
      ]
    }
  ],
  "errorCount": 1
}
```

#### GET /health

헬스 체크

**요청:**
```bash
curl http://localhost:8080/health
```

**응답:**
```json
{
  "status": "ok",
  "service": "kospell"
}
```

### 사용 예시

Python에서 사용:
```python
import requests
import json

response = requests.post(
    'http://localhost:8080/v1/check-spell',
    json={
        'text': '너는나와 kafka',
        'words': ['kafka']
    }
)
result = response.json()
print(f"오류 수: {result['errorCount']}")
for correction in result['corrections']:
    for item in correction['items']:
        print(f"{item['origin']} -> {item['suggest']}")
```

JavaScript/Node.js에서 사용:
```javascript
const response = await fetch('http://localhost:8080/v1/check-spell', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    text: '너는나와 kafka',
    words: ['kafka']
  })
});
const result = await response.json();
console.log(`오류 수: ${result.errorCount}`);
```

---

## Docker / Docker Compose

Dockerfile과 `docker-compose.yml`를 제공합니다.

### nara 모드 (기본)

`kospell/` 디렉터리에서:

```bash
docker compose up --build
```

기본 접속:

- http://localhost:8080/health
- http://localhost:8080/ (Redoc)

### local 모드 (hunspell)

1) Hunspell 사전 준비: `<lang>.aff` / `<lang>.dic`

예) `kospell/hunspell/ko.aff`, `kospell/hunspell/ko.dic`

2) 실행:

```bash
docker compose --profile local up --build
```

기본 포트는 `http://localhost:8081` 입니다.

### 환경변수 (선택)

- `KOSPELL_HTTP_PORT` (기본: `8080`) — nara 서비스 호스트 포트
- `KOSPELL_LOCAL_HTTP_PORT` (기본: `8081`) — local 서비스 호스트 포트
- `KOSPELL_HUNSPELL_DICT_PATH` (기본: `./hunspell`) — 사전 디렉터리 경로
- `KOSPELL_HUNSPELL_LANG` (기본: `ko`) — 사전 이름 (파일명이 `<lang>.aff/.dic` 이어야 함)

## 주의사항

- 비상업적 용도로만 사용 가능
- API 요청은 병렬 처리되며, `GOMAXPROCS`에 의해 동시성 제어
- 장문은 자동으로 300 어절 단위로 분할 처리
- 네트워크 요청이므로 적절한 타임아웃 설정 필요 (권장: 8-10초)

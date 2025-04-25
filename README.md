# GBot - Discord Bot in Go

Python으로 작성된 Discord 봇을 Go 언어로 변환한 프로젝트입니다.

## 프로젝트 구조 (Project Structure)

```
gbot/
├── cmd/                      # 각 애플리케이션의 진입점
│   ├── hybabot/              # Discord 봇 메인
│   └── pricesota/            # 웹 크롤러 메인
├── configs/                  # 설정 파일
├── docker-compose.yml        # Docker Compose 설정
├── Dockerfile.bot            # 봇용 Dockerfile
├── Dockerfile.crawler        # 크롤러용 Dockerfile
├── go.mod                    # Go module file
├── go.sum                    # Go dependencies checksums
└── internal/                 # 내부 패키지
    ├── bot/                  # 봇 로직
    │   ├── bot.go            # 봇 구현 메인
    │   ├── commands/         # 명령어 핸들러
    │   ├── handlers/         # 이벤트 핸들러
    │   └── services/         # 비즈니스 로직 서비스
    ├── common/               # 공통 유틸리티
    ├── crawler/              # 웹 크롤러
    │   ├── crawler.go        # 크롤러 구현 메인
    │   ├── base_crawler.go   # 기본 크롤러 기능
    │   ├── discord.go        # 크롤러용 Discord 통합
    │   ├── parser/           # HTML 파서
    │   └── sources/          # 소스별 크롤러
    ├── models/               # 데이터 모델
    │   ├── keyword_alert.go  # 알림 모델
    │   ├── product.go        # 상품 모델
    │   └── food.go           # 음식 추천 모델
    └── storage/              # 데이터 저장소
        ├── mongodb.go        # MongoDB 연결
        ├── food_repository.go # 음식 저장소
        └── ...               # 기타 저장소
```

## 기능 (Features)

1. Discord Bot (`hybabot`)
   - 접두사를 사용한 명령어 처리
   - 상품 딜에 대한 알림 시스템
   - 음식 추천 시스템
   - 테스트용 ping 명령어

2. Web Crawler (`pricesota`)
   - 상품 정보를 위한 딜 웹사이트 크롤링
   - 사용자 키워드 기반 Discord 알림 전송
   - 다양한 소스 지원 (Ppomppu, Quasarzone)

## 명령어 목록 (Command List)

- `!ping` - 봇 응답 시간 테스트
- `!alert add <keyword>` - 키워드 알림 추가
- `!alert remove <keyword>` - 키워드 알림 삭제
- `!alert list` - 알림 목록 보기
- `!점메추` - 랜덤 점심 메뉴 추천
- `!저메추` - 랜덤 저녁 메뉴 추천
- `!메뉴` - 모든 음식 옵션 목록
- `!점메추등록 <food>` - 점심 메뉴 등록
- `!저메추등록 <food>` - 저녁 메뉴 등록
- `!점메추삭제 <food>` - 점심 메뉴 삭제
- `!저메추삭제 <food>` - 저녁 메뉴 삭제

## 봇 실행하기 (Running the Bot)

### 요구사항 (Requirements)

- Go 1.21+
- MongoDB
- Discord Bot Token

### 환경 변수 (Environment Variables)

루트 디렉토리에 다음 변수를 포함한 `.env` 파일을 생성하세요:

```
DISCORD_TOKEN=your_discord_bot_token
MONGODB_URI=mongodb://localhost:27017
MONGODB_NAME=discord_bot
COMMAND_PREFIX=!
```

### 빌드 및 실행 (Building and Running)

```bash
# 봇 빌드
go build -o bin/bot ./cmd/hybabot

# 봇 실행
./bin/bot

# 크롤러 빌드
go build -o bin/crawler ./cmd/pricesota

# 크롤러 실행
./bin/crawler
```

### Docker

```bash
# 봇 빌드 및 실행
docker build -f Dockerfile.bot -t gbot/bot .
docker run --env-file .env gbot/bot

# 크롤러 빌드 및 실행
docker build -f Dockerfile.crawler -t gbot/crawler .
docker run --env-file .env gbot/crawler

# Docker Compose 사용
docker-compose up
```

## 개발 (Development)

### 새 명령어 추가하기 (Adding New Commands)

1. `internal/bot/commands/`에 새 파일 생성
2. `ping_command.go`의 패턴을 따라 명령어 핸들러 구현
3. `bot.go`에 명령어 등록

### 새 크롤러 소스 추가하기 (Adding New Crawler Sources)

1. `internal/crawler/sources/`에 새 파일 생성
2. `Source` 인터페이스 구현
3. `crawler.go`의 소스 목록에 추가
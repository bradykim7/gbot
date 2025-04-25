# Discord Bot & Web Crawler System

## 프로젝트 개요 (Project Overview)

이 프로젝트는 두 개의 주요 구성 요소를 가진 시스템입니다:

1. **Discord Bot (HybaBot)**: 사용자 명령어 처리 및 키워드 알림 등록 관리
2. **Web Crawler (PriceSota)**: 특가 사이트 크롤링 및 등록된 키워드와 일치하는 상품 발견 시 알림 전송

## 시스템 아키텍처 (System Architecture)

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│                 │    │                 │    │                 │
│   Discord Bot   │◄──►│     MongoDB     │◄──►│   Web Crawler   │
│    (HybaBot)    │    │                 │    │   (PriceSota)   │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
        ▲                                              │
        │                                              │
        │                                              ▼
┌─────────────────┐                        ┌─────────────────────┐
│                 │                        │                     │
│  Discord Users  │                        │   Deal Websites     │
│                 │                        │   (Ppomppu, etc.)   │
│                 │                        │                     │
└─────────────────┘                        └─────────────────────┘
```

### 핵심 기능 (Key Features)

1. **Discord Bot**:
   - 키워드 알림 등록/삭제/목록 보기
   - 음식 추천 (점심/저녁)
   - Ping 명령어를 통한 상태 확인

2. **Web Crawler**:
   - 여러 특가 사이트 동시 크롤링
   - MongoDB에 상품 데이터 저장
   - 등록된 키워드와 일치하는 상품 발견 시 Discord 알림 전송

## 개선된 기능 (Improved Features)

최근 개선된 기능들:

1. **향상된 크롤러 (Improved Crawler)**:
   - 멀티소스 병렬 크롤링
   - 강화된 오류 처리 및 재시도 로직
   - 자세한 통계 수집

2. **효율적인 알림 시스템 (Efficient Notification System)**:
   - 더 나은 키워드 매칭 알고리즘
   - 사용자 정보 처리 개선
   - 이미지 및 할인율 표시 지원

3. **확장된 데이터 모델 (Enhanced Data Models)**:
   - 통계 및 메타데이터 수집
   - 이미지 URL 및 할인율 지원
   - 알림 상태 추적

## 설치 방법 (Installation)

### 요구 사항 (Requirements)
- Go 1.16+
- MongoDB
- Discord Bot Token

### 환경 변수 설정 (Environment Variables)
```
DISCORD_TOKEN=your_discord_bot_token
COMMAND_PREFIX=!
MONGODB_URI=mongodb://localhost:27017/discord_bot
CRAWL_INTERVAL_MINUTES=30
PRODUCT_CHANNEL_ID=your_discord_channel_id
```

### 빌드 방법 (Build Instructions)
```bash
# Discord Bot 빌드
go build -o hybabot ./cmd/hybabot

# Web Crawler 빌드
go build -o pricesota ./cmd/pricesota
```

### Docker 실행 방법 (Docker Setup)
```bash
# Docker Compose로 모든 서비스 실행
docker-compose up -d
```

## 사용 가이드 (Usage Guide)

### Discord Bot 명령어 (Commands)
- `!ping` - 봇 응답 시간 확인
- `!alert add [키워드]` - 키워드 알림 추가
- `!alert remove [키워드]` - 키워드 알림 삭제
- `!alert list` - 알림 목록 보기
- `!메뉴 점심` - 점심 추천
- `!메뉴 저녁` - 저녁 추천

### 개발자 정보 (Developer Information)
이 프로젝트는 Python 버전에서 Go로 마이그레이션되었으며, 병렬 처리와 타입 안전성을 최대한 활용하도록 설계되었습니다.


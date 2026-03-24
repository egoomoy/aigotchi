# Aigotchi — Design Spec

AI 코딩 토큰 사용량으로 성장하는 터미널 다마고치.

## Overview

Claude Code / Codex CLI 사용 시 소비하는 토큰량을 추적하여, 터미널에서 키우는 캐릭터(aigotchi)가 성장하는 프로젝트. NFT 스타일의 랜덤 특성 조합으로 유니크한 캐릭터가 생성되며, 다마고치의 상태 관리·인터랙션 시스템을 갖춘다.

## Tech Stack

- **Language:** Go
- **TUI:** Bubbletea + Lipgloss (Charm 생태계)
- **CLI:** Cobra
- **렌더링:** ANSI 컬러 ASCII 아트
- **배포:** 단일 바이너리 (brew / GitHub releases)

## Architecture

File-Based + On-Demand TUI 아키텍처. 데몬 없이 파일 기반으로 상태를 관리하고, 실행할 때 시간 경과를 계산한다.

### Data Flow

```
Claude Code
  └─ Stop Hook ──▶ aigotchi collect
                       └─ transcript JSONL 파싱
                       └─ ~/.aigotchi/events.jsonl 에 append

aigotchi (TUI/CLI)
  └─ events.jsonl + state.json 읽기
  └─ 마지막 실행 이후 경과 시간 계산
  └─ 상태 업데이트 → 렌더링
```

### Data Store (`~/.aigotchi/`)

| File | Purpose |
|------|---------|
| `pet.json` | 캐릭터 정보 — 종류, 이름, 진화 단계, 특성 목록, 생성일, 시드 |
| `state.json` | 현재 상태 — 배고픔, 행복, 건강, XP, 마지막 업데이트 시각 |
| `events.jsonl` | 토큰 사용 이벤트 로그 — 타임스탬프, 토큰 수, 모델, 세션ID |
| `collect.json` | collector 상태 — 마지막 처리한 transcript 파일별 byte offset |

모든 파일은 `version` 필드를 포함하여 향후 마이그레이션을 지원한다.

### Data Schemas

**`pet.json`:**
```json
{
  "version": 1,
  "seed": 847291,
  "name": "Mochi",
  "stage": 4,
  "traits": ["coral", "happy_eyes", "crown", "electric"],
  "personality": "nerdy",
  "rare": null,
  "created_at": "2026-03-24T12:00:00Z"
}
```

**`state.json`:**
```json
{
  "version": 1,
  "hunger": 80,
  "happiness": 60,
  "health": 90,
  "xp": 12400,
  "total_tokens_earned": 12400000,
  "total_tokens_spent": 200000,
  "last_updated": "2026-03-24T18:30:00Z"
}
```

게이지 범위: 모두 0–100 (정수). 감소/회복 단위도 이 범위 기준.

**`events.jsonl` (한 줄씩):**
```json
{"ts": "2026-03-24T12:30:00Z", "tokens": 4523, "model": "opus", "session": "abc-123"}
```

**`collect.json`:**
```json
{
  "version": 1,
  "offsets": {
    "/Users/koo/.claude/projects/-Users-koo-codecode/abc-123.jsonl": 84720
  }
}
```

## Character System

### Evolution Stages (5단계)

| Stage | Name | XP Threshold | Traits |
|-------|------|-------------|--------|
| 1 | Egg | 0 | 없음 |
| 2 | Baby | 100K tokens | 특성 1개 |
| 3 | Junior | 1M tokens | 특성 2개 |
| 4 | Senior | 10M tokens | 특성 3개 |
| 5 | Legend | 100M tokens | 특성 4개 + 레어 |

### Trait Layers (NFT 스타일 조합)

진화할 때마다 랜덤 특성이 하나씩 추가된다. 시드는 첫 실행 시 결정되어 같은 사용자는 항상 같은 진화 경로를 가진다.

| Layer | Options | Count |
|-------|---------|-------|
| Body Color | Mint, Coral, Lavender, Gold, Crimson, Ice, Shadow, Neon | 8 |
| Eyes | (°_°), (◕‿◕), (⊙_⊙), (≖_≖), (◉‿◉), (￣▽￣) | 6 |
| Accessory | 모자, 안경, 망토, 왕관, 후드, 날개, 뿔 | 7 |
| Aura (Senior+) | Sparkle, Electric, Fire, Ice, Crystal | 5 |
| Personality | Chill, Hyper, Grumpy, Nerdy, Sleepy, Chaotic | 6 |
| Rare (Legend only) | Holographic, Glitch, Rainbow, Cosmic, Void | 5 |

진화 시 특성 추가 순서: Body Color (Baby) → Eyes (Junior) → Accessory (Senior) → Aura (Legend). Personality는 Baby 단계에서 Body Color과 함께 결정. Rare는 Legend 단계에서 Aura와 함께 결정.

총 유니크 조합 (Legend): 8 × 6 × 7 × 5 × 6 × 5 = **50,400가지**
(Senior 이하: 8 × 6 × 7 × 5 × 6 = 10,080가지)

### Seed System

시드는 `aigotchi init` 시 `crypto/rand`로 생성하는 64비트 정수. `pet.json`에 저장되며, 각 진화 단계에서 `fnv1a(seed + stage)`로 특성을 결정한다 (FNV-1a 해시, 버전 간 호환성 보장). `pet.json` 삭제 시 새로운 캐릭터가 생성됨 (의도적 — 리롤 허용).

## State System

### 3 Gauges

| Gauge | Decay | Recovery | At Zero |
|-------|-------|----------|---------|
| Hunger | 시간 경과 (6시간마다 -10) | `aigotchi feed` (XP 10 소비) | 건강 감소 가속 (3시간마다 -10) |
| Happiness | 시간 경과 (8시간마다 -10) | `aigotchi play` (미니게임, +30) | 캐릭터 우울 표정 |
| Health | Hunger 0일 때 가속 감소 (3시간마다 -10) | Hunger≥30 & Happiness≥30 시 자연 회복 (12시간마다 +5) | 진화 퇴화 (아래 참조) |

### De-evolution & Death

- Health가 0이 되면 한 단계 퇴화. 마지막으로 얻은 특성이 제거됨.
- 퇴화 후 Health는 50으로 리셋. XP는 유지됨 (재진화 가능).
- Egg 단계에서 Health 0 → "Dormant" 상태. 토큰 사용 이벤트가 들어오면 Health 50으로 부활.
- 영구 사망은 없음. 최악의 경우 Dormant Egg로 돌아감.

### XP Economy

- 토큰 사용량 → XP 전환: **1K tokens = 1 XP**
- 진화 임계값: Egg(0) → Baby(100 XP = 100K tokens) → Junior(1,000 XP = 1M tokens) → Senior(10,000 XP = 10M tokens) → Legend(100,000 XP = 100M tokens)
- 진화 조건: XP 임계값 도달 + Health 50 이상
- 진화 시 시드 기반으로 랜덤 특성 추가
- `feed` 비용: **XP 10** (= 10K tokens 분량). 일반적인 코딩 세션(~50K tokens)으로 5번 먹일 수 있음.
- XP는 `total_tokens_earned` (누적 획득)과 `total_tokens_spent` (먹이 소비)로 분리 관리. 진화 판정은 earned 기준, feed 가능 여부는 (earned - spent) 기준.

## CLI Interface

```
aigotchi              # 메인 TUI (인터랙티브, bubbletea)
aigotchi status       # 한줄 상태 출력 (agent-deck용)
aigotchi feed         # 먹이주기
aigotchi play         # 미니게임
aigotchi name <이름>  # 이름 짓기
aigotchi stats        # 토큰 사용 통계
aigotchi collect      # hook에서 호출 (토큰 수집)
aigotchi init         # 최초 설정 + hook 등록 (멱등 — 재실행 시 pet 보존, hook만 재등록)
```

### Quick Status 출력 포맷

`aigotchi status`는 다음 형식으로 stdout에 한 줄 출력:

```
[Sr] Mochi | H:██░ ☺:█░░ ♥:██░ | 12.4K xp
```

### TUI Screens

1. **메인 화면** — 캐릭터 ASCII 아트 + 상태 게이지 + 특성 + 키보드 단축키
2. **Quick Status** — `aigotchi status`로 한줄 출력 (agent-deck 패널용)
3. **진화 이벤트** — 진화 시 before/after 애니메이션
4. **미니게임** — `aigotchi play`로 실행. 간단한 타이핑 게임 (랜덤 코드 키워드를 빠르게 입력). 성공 시 Happiness +30, 실패해도 +10. 1회 약 30초 소요. Personality에 따라 게임 대사가 달라짐.

## Hook Integration

### Claude Code Hook 설정

`aigotchi init`이 `~/.claude/settings.json`에 자동 등록:

```json
{
  "hooks": {
    "Stop": [{
      "command": "aigotchi collect --session-id $SESSION_ID --cwd $CWD"
    }]
  }
}
```

Claude Code `Stop` hook은 환경변수로 `SESSION_ID`, `CWD`, `TRANSCRIPT_PATH` 등을 전달한다.

### Transcript JSONL Schema

Claude Code 세션 파일 (`~/.claude/projects/<project>/<session-id>.jsonl`) 의 assistant 메시지 구조:

```json
{
  "type": "assistant",
  "message": {
    "model": "claude-opus-4-6[1m]",
    "usage": {
      "input_tokens": 3,
      "cache_creation_input_tokens": 8260,
      "cache_read_input_tokens": 0,
      "output_tokens": 60
    }
  },
  "timestamp": "2026-03-24T12:30:00Z"
}
```

collector는 `type: "assistant"` 인 라인만 읽고, `message.usage`에서 `input_tokens + output_tokens + cache_creation_input_tokens + cache_read_input_tokens`를 합산한다.

### `aigotchi collect` 동작

1. `$SESSION_ID`와 `$CWD`로 세션 JSONL 파일 경로 결정
2. `collect.json`에서 해당 파일의 마지막 byte offset 읽기
3. offset 이후 라인만 파싱, assistant 메시지의 usage 합산
4. `~/.aigotchi/events.jsonl`에 append (O_APPEND 플래그로 atomic append)
5. `collect.json`에 새 offset 저장

### 에러 처리

- `aigotchi` 바이너리가 PATH에 없으면: hook이 조용히 실패. `aigotchi init`에서 PATH 검증.
- transcript 파일이 비었거나 malformed: 해당 라인 스킵, stderr에 경고 로그
- 디스크 풀: append 실패 시 stderr에 에러, 다음 collect에서 재시도 (offset 미갱신)
- `init` 전에 collect 호출: `~/.aigotchi/` 없으면 즉시 exit 0 (에러 없이)

### File Safety

- 모든 JSON 쓰기는 write-then-rename (임시 파일에 쓰고 rename) 패턴
- events.jsonl append는 O_APPEND 플래그로 atomic
- 병렬 Claude Code 세션에서 동시 collect 호출 시에도 events.jsonl append는 안전 (POSIX O_APPEND 보장)
- TUI가 읽는 동안 collect가 쓰는 경우: TUI는 읽기 전용이므로 충돌 없음. state.json은 TUI만 업데이트.

### 데이터 소스

- **Primary:** Claude Code `Stop` hook → 세션 JSONL 파싱 (실시간)
- **Bootstrap:** `~/.claude/stats-cache.json` — `aigotchi init` 시 기존 사용량으로 초기 XP 부여 (선택적)
- Codex CLI 지원은 향후 작업. 동일한 events.jsonl 포맷으로 확장 가능.

## Project Structure

```
aigotchi/
├── cmd/aigotchi/
│   └── main.go                  # CLI 엔트리포인트 (cobra)
├── internal/
│   ├── collector/collector.go   # transcript JSONL 파싱, 토큰 추출
│   ├── engine/
│   │   ├── state.go             # 상태 계산 (시간 경과, 감소/회복)
│   │   ├── evolution.go         # 진화 로직 + 특성 랜덤 생성
│   │   └── interaction.go       # feed, play 커맨드 처리
│   ├── pet/
│   │   ├── pet.go               # Pet 구조체 + 직렬화
│   │   └── traits.go            # 특성 정의
│   ├── renderer/
│   │   ├── ascii.go             # ANSI 컬러 ASCII 아트 렌더링
│   │   ├── templates/           # 진화 단계별 ASCII 템플릿
│   │   └── compose.go           # 특성 레이어 합성
│   ├── storage/storage.go       # ~/.aigotchi/ 파일 읽기/쓰기
│   └── tui/
│       ├── app.go               # bubbletea 메인 앱
│       ├── views.go             # 화면별 뷰
│       └── statusline.go        # quick status 한줄 출력
├── go.mod
├── go.sum
└── Makefile
```

## Testing Strategy

| Package | Approach |
|---------|----------|
| `engine/` | 단위 테스트 — 시간 경과 계산, 진화 임계값, 특성 시드 검증 |
| `collector/` | 통합 테스트 — 실제 Claude Code JSONL 샘플 사용 |
| `renderer/` | 스냅샷 테스트 — golden file 비교 |
| `tui/` | bubbletea 테스트 프레임워크 — 키 입력 시뮬레이션 |

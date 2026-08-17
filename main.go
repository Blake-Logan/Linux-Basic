package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ANSI 색상 및 터미널 스타일 상수 정의
const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
	ColorItalic = "\033[3m"

	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"

	BgBlue   = "\033[44m"
	BgPurple = "\033[45m"
	BgCyan   = "\033[46m"
)

// 플레이어 저장 데이터 구조체
type PlayerData struct {
	Level           int             `json:"level"`
	XP              int             `json:"xp"`
	CompletedQuests map[string]bool `json:"completed_quests"`
}

// 퀘스트 검증 유형
type QuestType int

const (
	QuestTypeQuiz     QuestType = iota // 명령어/정답 입력형
	QuestTypeFileCheck                 // 실제 파일/폴더 생성 및 상태 검증형
)

// 퀘스트 정보 구조체
type Quest struct {
	ID          string
	Chapter     string
	Title       string
	Description string
	Goal        string
	Type        QuestType
	Answer      string               // Quiz 형일 때 정답
	CheckFunc   func() (bool, string) // FileCheck 형일 때 검증 함수
	CommandHint string
	XPReward    int
}

var (
	player   PlayerData
	quests   []Quest
	savePath string
	reader   *bufio.Reader
)

// 화면 지우기
func clearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		fmt.Print("\033[H\033[2J")
	}
}

// 텍스트 중앙 정렬 / 패딩
func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text
}

// 상자 스타일 테두리 출력 함수
func printBox(title string, lines []string, color string) {
	width := 64
	fmt.Printf("%s┌%s┐%s\n", color, strings.Repeat("─", width-2), ColorReset)
	
	if title != "" {
		paddedTitle := fmt.Sprintf(" %s ", title)
		titleLine := centerText(paddedTitle, width-2)
		fmt.Printf("%s│%s%s%s│%s\n", color, ColorBold+ColorWhite, titleLine, color, ColorReset)
		fmt.Printf("%s├%s┤%s\n", color, strings.Repeat("─", width-2), ColorReset)
	}

	for _, line := range lines {
		// 한글 및 ANSI 코드를 고려한 간이 길이 계산
		printableLen := len([]rune(stripANSI(line)))
		padLen := width - 4 - printableLen
		if padLen < 0 {
			padLen = 0
		}
		fmt.Printf("%s│%s %s%s %s│%s\n", color, ColorReset, line, strings.Repeat(" ", padLen), color, ColorReset)
	}
	fmt.Printf("%s└%s┘%s\n", color, strings.Repeat("─", width-2), ColorReset)
}

// ANSI 코드 제거 유틸리티
func stripANSI(str string) string {
	var result []rune
	inEscape := false
	for _, r := range str {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result = append(result, r)
	}
	return string(result)
}

// 저장 파일 경로 가져오기
func initSavePath() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	savePath = filepath.Join(home, ".linux_quest_save.json")
}

// 진행 상황 저장
func saveProgress() {
	data, err := json.MarshalIndent(player, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(savePath, data, 0644)
}

// 진행 상황 불러오기
func loadProgress() {
	initSavePath()
	player = PlayerData{
		Level:           1,
		XP:              0,
		CompletedQuests: make(map[string]bool),
	}

	file, err := os.ReadFile(savePath)
	if err != nil {
		return // 신규 플레이어
	}

	_ = json.Unmarshal(file, &player)
	if player.CompletedQuests == nil {
		player.CompletedQuests = make(map[string]bool)
	}
}

// 플레이어 칭호 반환
func getTitle(level int) string {
	switch {
	case level >= 10:
		return ColorPurple + "👑 터미널의 신 (Root Lord)" + ColorReset
	case level >= 8:
		return ColorRed + "🔥 커널 마스터 (Kernel Master)" + ColorReset
	case level >= 6:
		return ColorYellow + "⚡ 쉘의 연금술사 (Shell Alchemist)" + ColorReset
	case level >= 4:
		return ColorCyan + "🗡️ 디렉터리 정복자 (Directory Conqueror)" + ColorReset
	case level >= 2:
		return ColorGreen + "🌱 터미널 모험가 (Terminal Explorer)" + ColorReset
	default:
		return ColorWhite + "👾 리눅스 초보 (Linux Novice)" + ColorReset
	}
}

// XP 추가 및 레벨업 체크
func addXP(amount int) bool {
	player.XP += amount
	neededXP := player.Level * 100
	leveledUp := false

	for player.XP >= neededXP {
		player.XP -= neededXP
		player.Level++
		neededXP = player.Level * 100
		leveledUp = true
	}
	saveProgress()
	return leveledUp
}

// 퀘스트 데이터 생성 함수
func initQuests() {
	quests = []Quest{
		// ---------------- CHAPTER 1 ----------------
		{
			ID:          "q1",
			Chapter:     "Chapter 1: 첫 걸음과 위치 파악",
			Title:       "1. 현재 어디에 있는가? (pwd)",
			Description: "리눅스는 마우스 대신 명령어로 위치를 찾아갑니다.\n'Print Working Directory'의 약자로, 현재 위치한 폴더(디렉터리) 경로를 출력하는 명령어입니다.",
			Goal:        "현재 디렉터리를 확인하는 3글자 명령어는 무엇일까요?",
			Type:        QuestTypeQuiz,
			Answer:      "pwd",
			CommandHint: "힌트: p_d (3글자)",
			XPReward:    50,
		},
		{
			ID:          "q2",
			Chapter:     "Chapter 1: 첫 걸음과 위치 파악",
			Title:       "2. 주변 탐색하기 (ls)",
			Description: "'List'의 약자로, 현재 폴더에 어떤 파일과 폴더가 있는지 목록을 보여줍니다.",
			Goal:        "숨김 파일(.으로 시작하는 파일)까지 포함하여 상세 정보를 보는 옵션 결합 명령어는 무엇일까요?",
			Type:        QuestTypeQuiz,
			Answer:      "ls -la",
			CommandHint: "힌트: ls 뒤에 -a(모두)와 -l(상세) 옵션을 붙여보세요. (예: ls -a -l 또는 ls -la)",
			XPReward:    60,
		},
		{
			ID:          "q3",
			Chapter:     "Chapter 1: 첫 걸음과 위치 파악",
			Title:       "3. 모험의 시작 - 폴더 생성 (mkdir 실습)",
			Description: "💡 [안내] 이 퀘스트는 게임 밖 터미널(새 세션)에서 직접 명령어를 쳐야 합니다!\n'Make Directory'의 약자로 폴더를 만드는 명령어입니다.",
			Goal:        "새 터미널 창을 열어 `mkdir quest_camp` 명령어를 실행한 후, 게임으로 돌아와 [검증하기]를 누르세요.",
			Type:        QuestTypeFileCheck,
			CheckFunc: func() (bool, string) {
				info, err := os.Stat("quest_camp")
				if err != nil {
					return false, "아직 'quest_camp' 폴더를 찾을 수 없습니다. (Termux 왼쪽을 스와이프해 새 창에서 `mkdir quest_camp`를 입력해보세요!)"
				}
				if !info.IsDir() {
					return false, "'quest_camp'가 디렉터리가 아니라 파일입니다! `rm quest_camp` 후 `mkdir quest_camp`를 실행하세요."
				}
				return true, "축하합니다! 'quest_camp' 디렉터리가 성공적으로 확인되었습니다."
			},
			CommandHint: "명령어: mkdir quest_camp",
			XPReward:    80,
		},
		{
			ID:          "q4",
			Chapter:     "Chapter 1: 첫 걸음과 위치 파악",
			Title:       "4. 이동의 기술 (cd)",
			Description: "'Change Directory'의 약자로 폴더를 이동할 때 씁니다.\n리눅스에서 점 두 개(..)는 '바로 위 폴더'를 의미합니다.",
			Goal:        "바로 한 단계 상위 디렉터리(부모 폴더)로 이동할 때 사용하는 명령어는 무엇일까요?",
			Type:        QuestTypeQuiz,
			Answer:      "cd ..",
			CommandHint: "힌트: cd 명령어 뒤에 띄어쓰기 한 번 하고 마침표 두 개(..)를 붙입니다.",
			XPReward:    60,
		},

		// ---------------- CHAPTER 2 ----------------
		{
			ID:          "q5",
			Chapter:     "Chapter 2: 파일과 디렉터리 조작",
			Title:       "5. 새로운 문서 만들기 (touch 실습)",
			Description: "💡 [안내] 터미널 새 창에서 실습하세요!\n빈 파일이나 문서를 생성하는 가장 간단한 방법은 touch 명령어입니다.",
			Goal:        "'quest_camp' 폴더 안이나 현재 위치에 'hero.txt' 라는 빈 파일을 생성하세요.",
			Type:        QuestTypeFileCheck,
			CheckFunc: func() (bool, string) {
				_, err := os.Stat("hero.txt")
				if err != nil {
					// quest_camp 내부도 확인
					_, err2 := os.Stat("quest_camp/hero.txt")
					if err2 != nil {
						return false, "'hero.txt' 파일을 찾지 못했습니다. `touch hero.txt`를 실행해보세요."
					}
				}
				return true, "'hero.txt' 파일이 성공적으로 생성되었습니다!"
			},
			CommandHint: "명령어: touch hero.txt",
			XPReward:    80,
		},
		{
			ID:          "q6",
			Chapter:     "Chapter 2: 파일과 디렉터리 조작",
			Title:       "6. 데이터 적어 넣기 (echo & redirection)",
			Description: "💡 [안내] 터미널 새 창에서 실습하세요!\n에코(echo)는 메아리처럼 글자를 출력하고, '>' 기호는 그 글자를 파일에 덮어씁니다.",
			Goal:        "echo 명령어와 '>' 기호를 이용해 방금 만든 'hero.txt'에 아무 글자나 적어보세요.",
			Type:        QuestTypeFileCheck,
			CheckFunc: func() (bool, string) {
				data, err := os.ReadFile("hero.txt")
				if err != nil {
					data, err = os.ReadFile("quest_camp/hero.txt")
					if err != nil {
						return false, "'hero.txt' 파일을 읽을 수 없습니다."
					}
				}
				if len(strings.TrimSpace(string(data))) == 0 {
					return false, "'hero.txt' 파일 내용이 비어있습니다. `echo \"안녕\" > hero.txt` 형식으로 적어보세요."
				}
				// ⚠️ 컴파일 에러의 원인이었던 콤마(,) 제거 완료
				return true, fmt.Sprintf("파일 내용 확인됨: [%s]", strings.TrimSpace(string(data)))
			},
			CommandHint: "예시: echo \"I am a Linux Hero\" > hero.txt",
			XPReward:    90,
		},
		{
			ID:          "q7",
			Chapter:     "Chapter 2: 파일과 디렉터리 조작",
			Title:       "7. 분신술 - 복사하기 (cp)",
			Description: "💡 [안내] 터미널 새 창에서 실습하세요!\n'Copy'의 약자로, 원본을 남겨두고 똑같은 복사본을 만듭니다.",
			Goal:        "'hero.txt' 파일을 'hero_backup.txt'라는 이름으로 복사하세요.",
			Type:        QuestTypeFileCheck,
			CheckFunc: func() (bool, string) {
				_, err := os.Stat("hero_backup.txt")
				if err != nil {
					return false, "'hero_backup.txt' 파일이 없습니다. `cp hero.txt hero_backup.txt`를 실행해보세요."
				}
				return true, "복사본 'hero_backup.txt' 생성 확인 완료!"
			},
			CommandHint: "명령어: cp hero.txt hero_backup.txt",
			XPReward:    80,
		},
		{
			ID:          "q8",
			Chapter:     "Chapter 2: 파일과 디렉터리 조작",
			Title:       "8. 완전 삭제의 위험성 (rm)",
			Description: "'Remove'의 약자입니다. 리눅스에는 휴지통이 없어서 지우면 복구가 어렵습니다!\n특히 폴더를 지울 때는 특별한 옵션이 필요합니다.",
			Goal:        "디렉터리와 그 내부의 모든 내용을 강제로 삭제하는 옵션 조합 명령은 무엇일까요?",
			Type:        QuestTypeQuiz,
			Answer:      "rm -rf",
			CommandHint: "힌트: rm 명령어 뒤에 -r(recursive: 하위폴더까지)과 -f(force: 강제로)를 붙입니다.",
			XPReward:    70,
		},

		// ---------------- CHAPTER 3 ----------------
		{
			ID:          "q9",
			Chapter:     "Chapter 3: 파일 내용 조회 및 탐색",
			Title:       "9. 파일 내용 출력하기 (cat)",
			Description: "파일의 모든 내용을 터미널 화면에 한 번에 쭉 보여주는 가장 기초적인 명령어입니다.",
			Goal:        "파일 내용을 바로 보여주는 3글자 명령어는?",
			Type:        QuestTypeQuiz,
			Answer:      "cat",
			CommandHint: "힌트: 고양이(Cat)와 철자가 같습니다.",
			XPReward:    50,
		},
		{
			ID:          "q10",
			Chapter:     "Chapter 3: 파일 내용 조회 및 탐색",
			Title:       "10. 보물찾기 - 텍스트 검색 (grep)",
			Description: "엄청나게 긴 텍스트 속에서 내가 원하는 단어나 패턴이 있는 줄만 쏙 뽑아주는 마법 같은 도구입니다.",
			Goal:        "파일 내에서 원하는 문자열을 검색하는 대표적인 리눅스 명령어는 무엇일까요?",
			Type:        QuestTypeQuiz,
			Answer:      "grep",
			CommandHint: "힌트: g_ep (4글자)",
			XPReward:    80,
		},
		{
			ID:          "q11",
			Chapter:     "Chapter 3: 파일 내용 조회 및 탐색",
			Title:       "11. 파이프라인의 마법 (|)",
			Description: "앞 명령어의 결과를 뒤 명령어의 재료(입력)로 넘겨주는 연결 통로입니다. (예: ls -la | grep hero)",
			Goal:        "키보드의 Enter 키 위(Shift + 역슬래시 ₩)에 있는 세로 막대기 기호는 무엇일까요?",
			Type:        QuestTypeQuiz,
			Answer:      "|",
			CommandHint: "힌트: '|' (Pipe) 기호 자체를 입력하세요.",
			XPReward:    70,
		},

		// ---------------- CHAPTER 4 ----------------
		{
			ID:          "q12",
			Chapter:     "Chapter 4: 권한과 관리자 권한",
			Title:       "12. 파일 권한 변경 (chmod 실습)",
			Description: "💡 [안내] 터미널 새 창에서 실습하세요!\n리눅스 파일에는 읽기(r), 쓰기(w), 실행(x) 권한이 있습니다. 권한을 바꾸려면 'Change Mode'를 뜻하는 chmod를 씁니다.",
			Goal:        "'hero.txt' 파일의 권한을 모든 사용자가 읽기/쓰기/실행 할수 있게(777) 변경해보세요.",
			Type:        QuestTypeFileCheck,
			CheckFunc: func() (bool, string) {
				info, err := os.Stat("hero.txt")
				if err != nil {
					return false, "'hero.txt' 파일이 없습니다. 이전 퀘스트를 참고하여 터미널에서 `touch hero.txt`로 만들어주세요."
				}
				mode := info.Mode().Perm()
				// 0777 이거나 실행 권한이 있는지 체크
				if mode == 0777 || (mode&0111 != 0) {
					return true, fmt.Sprintf("권한 변경 확인 성공! 현재 퍼미션 모드: %o", mode)
				}
				return false, fmt.Sprintf("현재 권한은 %o 입니다. 터미널 창에서 `chmod 777 hero.txt` 를 실행해보세요.", mode)
			},
			CommandHint: "명령어: chmod 777 hero.txt",
			XPReward:    100,
		},
		{
			ID:          "q13",
			Chapter:     "Chapter 4: 권한과 관리자 권한",
			Title:       "13. 실행 중인 프로세스 확인 (ps)",
			Description: "현재 스마트폰(또는 PC)이 내부적으로 어떤 프로그램(프로세스)들을 돌리고 있는지 상태를 조회합니다.",
			Goal:        "모든 사용자의 실행 중인 프로세스를 상세히 보기 위해 자주 함께 쓰는 대표적인 옵션 조합(ps ___)은?",
			Type:        QuestTypeQuiz,
			Answer:      "ps aux",
			CommandHint: "힌트: a, u, x 옵션을 합친 'ps aux' (띄어쓰기 포함)",
			XPReward:    80,
		},

		// ---------------- CHAPTER 5 ----------------
		{
			ID:          "q14",
			Chapter:     "Chapter 5: Termux & 고급 스킬",
			Title:       "14. 패키지 설치하기 (pkg / apt)",
			Description: "구글 플레이스토어처럼 리눅스에도 앱스토어 역할을 하는 패키지 매니저가 있습니다.",
			Goal:        "Termux 전용 패키지 관리자 명령어(pkg)로 최신 프로그램 목록을 업데이트하는 명령어는?",
			Type:        QuestTypeQuiz,
			Answer:      "pkg update",
			CommandHint: "힌트: pkg u______ (띄어쓰기 포함)",
			XPReward:    90,
		},
		{
			ID:          "q15",
			Chapter:     "Chapter 5: Termux & 고급 스킬",
			Title:       "15. 나만의 스크립트 작성 (Shell Script)",
			Description: "명령어들을 파일 하나에 모아서 자동으로 촤르륵 실행하게 해주는 것을 '쉘 스크립트'라고 합니다.",
			Goal:        "Bash 쉘 스크립트를 만들 때 맨 첫 줄에 적는 '#!/bin/____' 에서 빈칸에 들어갈 4글자는?",
			Type:        QuestTypeQuiz,
			Answer:      "bash",
			CommandHint: "힌트: b_sh",
			XPReward:    100,
		},
	}
}

// 헤더 상자 출력
func renderHeader() {
	title := "⚔️  LINUX QUEST : TERMUX EDITION  ⚔️"
	
	completedCount := 0
	for _, q := range quests {
		if player.CompletedQuests[q.ID] {
			completedCount++
		}
	}

	progressPercent := float64(completedCount) / float64(len(quests)) * 100
	barWidth := 20
	filled := int(progressPercent / 100 * float64(barWidth))
	progressBar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	lines := []string{
		fmt.Sprintf("플레이어 칭호: %s", getTitle(player.Level)),
		fmt.Sprintf("레벨: %sLv.%d%s | XP: %s%d / %d%s", 
			ColorYellow, player.Level, ColorReset,
			ColorCyan, player.XP, player.Level*100, ColorReset),
		fmt.Sprintf("진행도: [%s%s%s] %.1f%% (%d/%d 완료)", 
			ColorGreen, progressBar, ColorReset, progressPercent, completedCount, len(quests)),
	}

	printBox(title, lines, ColorCyan)
}

// 초보자 가이드 화면 (새로 추가됨)
func showGuide() {
	clearScreen()
	lines := []string{
		"환영합니다! 이 게임은 단순 퀴즈가 아닌 실제 리눅스 환경과 연동됩니다.",
		"",
		" [ 퀘스트 진행 방법 ]",
		" 1. 퀴즈(Quiz) 퀘스트 : 게임 내에서 바로 정답 명령어를 입력하세요.",
		" 2. 실습(Check) 퀘스트: 게임 밖 '실제 터미널'에서 명령어를 쳐야 합니다!",
		"",
		" [ 💡 실습 퀘스트 꿀팁 - Termux 사용자 ]",
		" - 화면 왼쪽 가장자리를 오른쪽으로 스와이프하여 [NEW SESSION]을 누르세요.",
		" - 새 창에서 명령어를 연습 및 실행하고, 다시 기존 창(게임)으로 돌아와서",
		"   [실습 완료 검증하기]를 누르면 아주 편리합니다!",
		"",
		" - 게임을 종료(0번)하고 실습한 뒤 다시 켜도 여러분의 레벨은 저장됩니다.",
	}
	printBox("📖 초보자 가이드 (How to Play)", lines, ColorGreen)

	fmt.Println("\n엔터를 누르면 메인 메뉴로 돌아갑니다...")
	reader.ReadString('\n')
}

// 메인 메뉴
func showMainMenu() {
	for {
		clearScreen()
		renderHeader()

		fmt.Println()
		fmt.Printf(" %s[1]%s 📜 퀘스트 목록 (Quest Log)\n", ColorYellow, ColorReset)
		fmt.Printf(" %s[2]%s 👤 나의 상태 및 칭호 (Profile)\n", ColorYellow, ColorReset)
		fmt.Printf(" %s[3]%s 📖 초보자 가이드 (How to Play)\n", ColorYellow, ColorReset)
		fmt.Printf(" %s[4]%s 🔄 게임 초기화 (Reset Progress)\n", ColorYellow, ColorReset)
		fmt.Printf(" %s[0]%s 🚪 모험 종료 (Exit)\n", ColorYellow, ColorReset)
		fmt.Println()
		fmt.Printf("%s선택할 메뉴 번호를 입력하세요 > %s", ColorBold+ColorCyan, ColorReset)

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			showQuestList()
		case "2":
			showProfile()
		case "3":
			showGuide()
		case "4":
			resetProgress()
		case "0":
			clearScreen()
			fmt.Printf("\n%s👋 리눅스 모험을 이용해 주셔서 감사합니다! 다음에 또 만나요!%s\n\n", ColorBold+ColorGreen, ColorReset)
			os.Exit(0)
		}
	}
}

// 퀘스트 목록 화면
func showQuestList() {
	for {
		clearScreen()
		fmt.Printf("%s=== 📜 퀘스트 목록 (QUEST LIST) ===%s\n\n", ColorBold+ColorYellow, ColorReset)

		currentChapter := ""
		for idx, q := range quests {
			if q.Chapter != currentChapter {
				currentChapter = q.Chapter
				fmt.Printf("\n%s%s%s\n", ColorBold+ColorCyan, currentChapter, ColorReset)
				fmt.Println(strings.Repeat("─", 50))
			}

			status := ColorRed + "🔒 [미완료]" + ColorReset
			if player.CompletedQuests[q.ID] {
				status = ColorGreen + "✅ [완  료]" + ColorReset
			}

			fmt.Printf(" %2d. %s %s (+%d XP)\n", idx+1, status, q.Title, q.XPReward)
		}

		fmt.Println()
		fmt.Printf("%s도전할 퀘스트 번호를 입력하세요 (0: 뒤로가기) > %s", ColorBold+ColorCyan, ColorReset)
		
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "0" {
			return
		}

		num, err := strconv.Atoi(input)
		if err == nil && num >= 1 && num <= len(quests) {
			playQuest(quests[num-1])
		}
	}
}

// 개별 퀘스트 실행/도전 화면
func playQuest(q Quest) {
	for {
		clearScreen()

		statusStr := ColorRed + "미완료" + ColorReset
		if player.CompletedQuests[q.ID] {
			statusStr = ColorGreen + "완료됨" + ColorReset
		}

		lines := []string{
			fmt.Sprintf("카테고리: %s", q.Chapter),
			fmt.Sprintf("상  태  : %s | 보상: %s+%d XP%s", statusStr, ColorYellow, q.XPReward, ColorReset),
			"",
			fmt.Sprintf("%s[퀘스트 설명]%s", ColorBold, ColorReset),
			q.Description,
			"",
			fmt.Sprintf("%s[목표]%s", ColorBold+ColorYellow, ColorReset),
			q.Goal,
		}

		printBox(fmt.Sprintf("QUEST: %s", q.Title), lines, ColorYellow)

		fmt.Println()
		if q.Type == QuestTypeQuiz {
			fmt.Printf(" %s[1]%s ✏️ 정답 입력하기\n", ColorYellow, ColorReset)
		} else {
			fmt.Printf(" %s[1]%s 🔍 실습 완료 검증하기 (Check)\n", ColorYellow, ColorReset)
		}
		fmt.Printf(" %s[2]%s 💡 힌트 보기 (Hint)\n", ColorYellow, ColorReset)
		fmt.Printf(" %s[0]%s 🔙 목록으로 돌아가기\n", ColorYellow, ColorReset)
		fmt.Println()
		fmt.Printf("%s선택 입력 > %s", ColorBold+ColorCyan, ColorReset)

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			if q.Type == QuestTypeQuiz {
				fmt.Printf("\n%s정답 명령어를 입력하세요: %s", ColorBold, ColorReset)
				ans, _ := reader.ReadString('\n')
				ans = strings.TrimSpace(ans)

				// 공백 무시 비교 지원
				if strings.EqualFold(strings.ReplaceAll(ans, " ", ""), strings.ReplaceAll(q.Answer, " ", "")) || ans == q.Answer {
					fmt.Printf("\n%s🎉 정답입니다! 퀘스트를 성공적으로 수행했습니다!%s\n", ColorBold+ColorGreen, ColorReset)
					if !player.CompletedQuests[q.ID] {
						player.CompletedQuests[q.ID] = true
						if addXP(q.XPReward) {
							fmt.Printf("%s✨ 레벨 업! 축하합니다! Level %d 달성!%s\n", ColorBold+ColorYellow, player.Level, ColorReset)
						}
					}
					time.Sleep(2 * time.Second)
					return
				} else {
					fmt.Printf("\n%s❌ 아쉽게도 틀렸습니다. 다시 시도해보세요!%s\n", ColorRed, ColorReset)
					time.Sleep(1500 * time.Millisecond)
				}
			} else if q.Type == QuestTypeFileCheck {
				fmt.Println("\n🔎 시스템 검증 실행 중...")
				time.Sleep(500 * time.Millisecond)

				success, msg := q.CheckFunc()
				if success {
					fmt.Printf("\n%s🎉 %s%s\n", ColorBold+ColorGreen, msg, ColorReset)
					if !player.CompletedQuests[q.ID] {
						player.CompletedQuests[q.ID] = true
						if addXP(q.XPReward) {
							fmt.Printf("%s✨ 레벨 업! 축하합니다! Level %d 달성!%s\n", ColorBold+ColorYellow, player.Level, ColorReset)
						}
					}
					time.Sleep(2 * time.Second)
					return
				} else {
					fmt.Printf("\n%s❌ 검증 실패: %s%s\n", ColorRed, msg, ColorReset)
					fmt.Println("엔터를 누르면 계속합니다...")
					reader.ReadString('\n')
				}
			}
		case "2":
			fmt.Printf("\n%s💡 %s%s\n", ColorYellow, q.CommandHint, ColorReset)
			fmt.Println("엔터를 누르면 계속합니다...")
			reader.ReadString('\n')
		case "0":
			return
		}
	}
}

// 내 프로필 및 통계
func showProfile() {
	clearScreen()
	completedCount := 0
	for _, q := range quests {
		if player.CompletedQuests[q.ID] {
			completedCount++
		}
	}

	lines := []string{
		fmt.Sprintf("현재 레벨 : Lv.%d", player.Level),
		fmt.Sprintf("획득 칭호 : %s", getTitle(player.Level)),
		fmt.Sprintf("보유 XP   : %d / %d", player.XP, player.Level*100),
		fmt.Sprintf("클리어한 퀘스트 : %d / %d 개", completedCount, len(quests)),
		fmt.Sprintf("저장 데이터 위치 : %s", savePath),
	}

	printBox("👤 모험가 프로필", lines, ColorPurple)

	fmt.Println("\n엔터를 누르면 메인 메뉴로 돌아갑니다...")
	reader.ReadString('\n')
}

// 게임 진행 상황 초기화
func resetProgress() {
	clearScreen()
	fmt.Printf("%s⚠️ 정말로 모든 진행 상황을 초기화하시겠습니까? (y/N) > %s", ColorBold+ColorRed, ColorReset)
	
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" {
		player = PlayerData{
			Level:           1,
			XP:              0,
			CompletedQuests: make(map[string]bool),
		}
		saveProgress()
		fmt.Printf("\n%s🔄 초기화되었습니다! 처음부터 다시 도전해보세요.%s\n", ColorGreen, ColorReset)
		time.Sleep(1500 * time.Millisecond)
	}
}

// 메인 루프 실행
func main() {
	reader = bufio.NewReader(os.Stdin)
	initQuests()
	loadProgress()
	showMainMenu()
}

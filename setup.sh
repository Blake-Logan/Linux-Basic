#!/usr/bin/env bash
set -e

# ANSI Color Codes
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m' # No Color

print_banner() {
    echo -e "${CYAN}──────────────────────────────────────────────────────────${NC}"
    echo -e "${BOLD}${YELLOW}   ⚔️  LINUX QUEST : AUTOMATED SETUP & INSTALLER  ⚔️${NC}"
    echo -e "${CYAN}──────────────────────────────────────────────────────────${NC}"
}

detect_environment() {
    IS_TERMUX=false
    if [ -n "$TERMUX_VERSION" ] || [[ "$PREFIX" == *"com.termux"* ]]; then
        IS_TERMUX=true
        echo -e "${GREEN}✓ Detected Environment: ${BOLD}Termux (Android)${NC}"
    else
        OS_NAME=$(uname -s)
        echo -e "${GREEN}✓ Detected Environment: ${BOLD}Linux ($OS_NAME)${NC}"
    fi
}

install_golang() {
    if command -v go &>/dev/null; then
        echo -e "${GREEN}✓ Go compiler is already installed: ${CYAN}$(go version)${NC}"
        return 0
    fi

    echo -e "\n${YELLOW}➜ Go language is not installed. Installing automatically...${NC}"

    if [ "$IS_TERMUX" = true ]; then
        echo -e "${CYAN}➜ Running: pkg update -y && pkg install golang -y${NC}"
        pkg update -y && pkg install golang -y
    elif command -v apt-get &>/dev/null; then
        echo -e "${CYAN}➜ Detected Debian/Ubuntu. Running: apt update && apt install golang-go${NC}"
        if [ "$EUID" -ne 0 ]; then
            sudo apt-get update -y && sudo apt-get install -y golang-go
        else
            apt-get update -y && apt-get install -y golang-go
        fi
    elif command -v pacman &>/dev/null; then
        echo -e "${CYAN}➜ Detected Arch Linux. Running: pacman -Sy go${NC}"
        if [ "$EUID" -ne 0 ]; then
            sudo pacman -Sy --noconfirm go
        else
            pacman -Sy --noconfirm go
        fi
    elif command -v dnf &>/dev/null; then
        echo -e "${CYAN}➜ Detected Fedora/RHEL. Running: dnf install golang${NC}"
        if [ "$EUID" -ne 0 ]; then
            sudo dnf install -y golang
        else
            dnf install -y golang
        fi
    elif command -v zypper &>/dev/null; then
        echo -e "${CYAN}➜ Detected openSUSE. Running: zypper install go${NC}"
        if [ "$EUID" -ne 0 ]; then
            sudo zypper install -y go
        else
            zypper install -y go
        fi
    else
        echo -e "${RED}✗ Unsupported package manager. Please install Go manually.${NC}"
        exit 1
    fi

    echo -e "${GREEN}✓ Go language installed successfully!${NC}"
}

build_game() {
    echo -e "\n${YELLOW}➜ Compiling 'main.go' into 'helper' executable binary...${NC}"

    if [ ! -f "main.go" ]; then
        echo -e "${RED}✗ Error: 'main.go' file not found in current directory!${NC}"
        echo -e "${RED}  Please place 'setup.sh' in the same folder as 'main.go'.${NC}"
        exit 1
    fi

    # Compile Go code into helper binary
    go build -o helper main.go

    if [ -f "helper" ]; then
        echo -e "${GREEN}✓ Compilation successful! Output: ./helper${NC}"
    else
        echo -e "${RED}✗ Compilation failed.${NC}"
        exit 1
    fi
}

install_binary() {
    echo -e "\n${YELLOW}➜ Installing 'helper' binary to system PATH...${NC}"

    if [ "$IS_TERMUX" = true ]; then
        TARGET_DIR="$PREFIX/bin"
        cp helper "$TARGET_DIR/helper"
        chmod +x "$TARGET_DIR/helper"
        echo -e "${GREEN}✓ Installed to Termux binary directory: ${CYAN}$TARGET_DIR/helper${NC}"
    else
        # Standard Linux path resolution
        if [ -w "/usr/local/bin" ]; then
            TARGET_DIR="/usr/local/bin"
            cp helper "$TARGET_DIR/helper"
            chmod +x "$TARGET_DIR/helper"
        elif command -v sudo &>/dev/null; then
            TARGET_DIR="/usr/local/bin"
            sudo cp helper "$TARGET_DIR/helper"
            sudo chmod +x "$TARGET_DIR/helper"
        else
            TARGET_DIR="$HOME/.local/bin"
            mkdir -p "$TARGET_DIR"
            cp helper "$TARGET_DIR/helper"
            chmod +x "$TARGET_DIR/helper"

            # Add to path if not present
            if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
                echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
                echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc 2>/dev/null || true
            fi
        fi
        echo -e "${GREEN}✓ Installed binary to: ${CYAN}$TARGET_DIR/helper${NC}"
    fi
}

main() {
    clear
    print_banner
    detect_environment
    install_golang
    build_game
    install_binary

    echo -e "\n${CYAN}──────────────────────────────────────────────────────────${NC}"
    echo -e "${BOLD}${GREEN}🎉 설치가 완료되었습니다!${NC}"
    echo -e "${YELLOW}이제 터미널 어느 위치에서든 아래 명령어를 입력하여 실행할 수 있습니다:${NC}"
    echo -e "\n    ${BOLD}${CYAN}helper${NC}\n"
    echo -e "${CYAN}──────────────────────────────────────────────────────────${NC}"
}

main


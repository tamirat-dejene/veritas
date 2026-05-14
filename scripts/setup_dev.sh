#!/bin/bash

# setup_dev.sh - Automated local development environment setup for Veritas

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Starting local development setup...${NC}"

# Check dependencies
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}Error: python3 is not installed.${NC}"
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: go is not installed.${NC}"
    exit 1
fi

# 1. Setup Go Shared & Services
echo -e "${BLUE}Setting up Go dependencies...${NC}"
(cd shared && go mod tidy)
for service in services/*; do
    if [ -f "$service/go.mod" ]; then
        echo "Tidying $service..."
        (cd "$service" && go mod tidy)
    fi
done

# 2. Setup Python Services
echo -e "${BLUE}Setting up Python virtual environments...${NC}"
for service in services/*; do
    if [ -f "$service/requirements.txt" ]; then
        echo -e "Setting up ${GREEN}$(basename "$service")${NC}..."
        (
            cd "$service"
            if [ ! -d ".venv" ]; then
                python3 -m venv .venv
            fi
            source .venv/bin/activate
            pip install --upgrade pip
            pip install -r requirements.txt
            if [ -f "requirements-dev.txt" ]; then
                pip install -r requirements-dev.txt
            fi
            deactivate
        )
    fi
done

echo -e "${GREEN}Setup complete!${NC}"
echo -e "To start developing in a Python service, run: ${BLUE}source services/<service-name>/.venv/bin/activate${NC}"

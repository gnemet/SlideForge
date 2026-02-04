#!/bin/bash

# 🛡️ Digital Blacksmith: AI Bridge Verifier
# Standardized script to verify Gemini API access across the ecosystem.

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🛠️  Verifying Gemini AI Bridge...${NC}"

# 1. Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Error: Go is not installed.${NC}"
    exit 1
fi

# 2. Check if .env exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠️  Warning: .env not found. Attempting to use encrypted vault...${NC}"
    if [ -f .env.gpg ]; then
        ./scripts/vault.sh unlock .env
    else
        echo -e "${RED}❌ Error: No .env or .env.gpg found.${NC}"
        exit 1
    fi
fi

# 3. Running the Go test program
echo -e "🚀 Running Gemini diagnostic..."

# Check which test file exists
if [ -f scripts/test_gemini.go ]; then
    go run scripts/test_gemini.go
elif [ -f scripts/test_gemini_sf.go ]; then
    go run scripts/test_gemini_sf.go
else
    echo -e "${RED}❌ Error: Gemini test program not found in scripts/${NC}"
    exit 1
fi

RESULT=$?

if [ $RESULT -eq 0 ]; then
    echo -e "\n${GREEN}✨ SUCCESS: Gemini AI Bridge is operational and authenticated.${NC}"
else
    echo -e "\n${RED}❌ FAILURE: Gemini diagnostic failed. Check your API key and connection.${NC}"
fi

exit $RESULT

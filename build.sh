#!/bin/bash

# Build script for MineMock
# Cross-compiles for multiple platforms

set -e

APP_NAME="minemock"
SOURCE="minemock.go"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Building MineMock...${NC}"

# Clean old binaries to force rebuild
echo -e "${YELLOW}Cleaning old binaries...${NC}"
rm -f "${APP_NAME}" "${APP_NAME}_linux_amd64" "${APP_NAME}_linux_arm64"
rm -f "${APP_NAME}_windows_amd64.exe" "${APP_NAME}_windows_arm64.exe"
rm -f "${APP_NAME}_darwin_amd64" "${APP_NAME}_darwin_arm64"

# Current platform
echo -e "${GREEN}Building for current platform...${NC}"
go build -a -o "${APP_NAME}" "${SOURCE}"
echo -e "${GREEN}✓ ${APP_NAME}${NC}"

# Linux AMD64
echo -e "${GREEN}Building for Linux AMD64...${NC}"
GOOS=linux GOARCH=amd64 go build -a -o "${APP_NAME}_linux_amd64" "${SOURCE}"
echo -e "${GREEN}✓ ${APP_NAME}_linux_amd64${NC}"

# Linux ARM64
echo -e "${GREEN}Building for Linux ARM64...${NC}"
GOOS=linux GOARCH=arm64 go build -a -o "${APP_NAME}_linux_arm64" "${SOURCE}"
echo -e "${GREEN}✓ ${APP_NAME}_linux_arm64${NC}"

# Windows AMD64
echo -e "${GREEN}Building for Windows AMD64...${NC}"
GOOS=windows GOARCH=amd64 go build -a -o "${APP_NAME}_windows_amd64.exe" "${SOURCE}"
echo -e "${GREEN}✓ ${APP_NAME}_windows_amd64.exe${NC}"

# Windows ARM64
echo -e "${GREEN}Building for Windows ARM64...${NC}"
GOOS=windows GOARCH=arm64 go build -a -o "${APP_NAME}_windows_arm64.exe" "${SOURCE}"
echo -e "${GREEN}✓ ${APP_NAME}_windows_arm64.exe${NC}"

# macOS AMD64
echo -e "${GREEN}Building for macOS AMD64...${NC}"
GOOS=darwin GOARCH=amd64 go build -a -o "${APP_NAME}_darwin_amd64" "${SOURCE}"
echo -e "${GREEN}✓ ${APP_NAME}_darwin_amd64${NC}"

# macOS ARM64
echo -e "${GREEN}Building for macOS ARM64...${NC}"
GOOS=darwin GOARCH=arm64 go build -a -o "${APP_NAME}_darwin_arm64" "${SOURCE}"
echo -e "${GREEN}✓ ${APP_NAME}_darwin_arm64${NC}"

echo ""
echo -e "${GREEN}All builds completed successfully!${NC}"
echo ""
echo "Binaries:"
ls -lh "${APP_NAME}"* 2>/dev/null | grep -v ".go" | grep -v ".sh"

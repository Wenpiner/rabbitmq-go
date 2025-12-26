#!/bin/bash

# 清理脚本
# 删除所有构建产物和临时文件

set -e

echo "========================================="
echo "Cleaning RabbitMQ-Go Project"
echo "========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 清理 Go 缓存
echo -e "${YELLOW}1. Cleaning Go cache...${NC}"
go clean -cache -testcache -modcache 2>/dev/null || true
echo -e "${GREEN}✓ Go cache cleaned${NC}"
echo ""

# 清理示例二进制文件
echo -e "${YELLOW}2. Cleaning example binaries...${NC}"
find examples -type f ! -name "*.go" ! -name "*.md" ! -name "go.mod" ! -name "go.sum" -delete 2>/dev/null || true
echo -e "${GREEN}✓ Example binaries cleaned${NC}"
echo ""

# 清理测试二进制文件
echo -e "${YELLOW}3. Cleaning test binaries...${NC}"
find . -name "*.test" -delete 2>/dev/null || true
echo -e "${GREEN}✓ Test binaries cleaned${NC}"
echo ""

# 清理覆盖率文件
echo -e "${YELLOW}4. Cleaning coverage files...${NC}"
find . -name "*.out" -delete 2>/dev/null || true
find . -name "coverage.txt" -delete 2>/dev/null || true
find . -name "coverage.html" -delete 2>/dev/null || true
find . -name "*.cover" -delete 2>/dev/null || true
echo -e "${GREEN}✓ Coverage files cleaned${NC}"
echo ""

# 清理临时文件
echo -e "${YELLOW}5. Cleaning temporary files...${NC}"
find . -name "*.tmp" -delete 2>/dev/null || true
find . -name "*.log" -delete 2>/dev/null || true
find . -name "*.bak" -delete 2>/dev/null || true
find . -name "*~" -delete 2>/dev/null || true
echo -e "${GREEN}✓ Temporary files cleaned${NC}"
echo ""

# 清理 macOS 文件
echo -e "${YELLOW}6. Cleaning macOS files...${NC}"
find . -name ".DS_Store" -delete 2>/dev/null || true
echo -e "${GREEN}✓ macOS files cleaned${NC}"
echo ""

# 清理主二进制文件
echo -e "${YELLOW}7. Cleaning main binary...${NC}"
rm -f main 2>/dev/null || true
echo -e "${GREEN}✓ Main binary cleaned${NC}"
echo ""

echo "========================================="
echo -e "${GREEN}✓ Cleaning complete!${NC}"
echo "========================================="
echo ""


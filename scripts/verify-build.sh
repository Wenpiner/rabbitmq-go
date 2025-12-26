#!/bin/bash

# 验证构建脚本
# 检查所有代码是否可以编译和测试

set -e

echo "========================================="
echo "RabbitMQ-Go v4.0.0 Build Verification"
echo "========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查 Go 版本
echo -e "${YELLOW}1. Checking Go version...${NC}"
go version
echo ""

# 清理
echo -e "${YELLOW}2. Cleaning...${NC}"
go clean -cache
echo -e "${GREEN}✓ Clean complete${NC}"
echo ""

# 下载依赖
echo -e "${YELLOW}3. Downloading dependencies...${NC}"
go mod download
echo -e "${GREEN}✓ Dependencies downloaded${NC}"
echo ""

# 验证依赖
echo -e "${YELLOW}4. Verifying dependencies...${NC}"
go mod verify
echo -e "${GREEN}✓ Dependencies verified${NC}"
echo ""

# 编译所有包
echo -e "${YELLOW}5. Building all packages...${NC}"
go build ./...
echo -e "${GREEN}✓ All packages built successfully${NC}"
echo ""

# 运行 logger 包测试
echo -e "${YELLOW}6. Testing logger package...${NC}"
go test ./logger/... -v
echo -e "${GREEN}✓ Logger tests passed${NC}"
echo ""

# 运行 tracing 包测试
echo -e "${YELLOW}7. Testing tracing package...${NC}"
go test ./tracing/... -v
echo -e "${GREEN}✓ Tracing tests passed${NC}"
echo ""

# 运行 conf 包测试
echo -e "${YELLOW}8. Testing conf package...${NC}"
go test ./conf/... -v
echo -e "${GREEN}✓ Conf tests passed${NC}"
echo ""

# 编译所有示例
echo -e "${YELLOW}9. Building all examples...${NC}"
for dir in examples/*/; do
    if [ -f "$dir/main.go" ]; then
        echo "  Building $dir..."
        (cd "$dir" && go build -o /dev/null .)
    fi
done
echo -e "${GREEN}✓ All examples built successfully${NC}"
echo ""

# 检查代码格式
echo -e "${YELLOW}10. Checking code format...${NC}"
if [ -n "$(gofmt -l .)" ]; then
    echo -e "${RED}✗ Code is not formatted${NC}"
    echo "Run: gofmt -w ."
    exit 1
else
    echo -e "${GREEN}✓ Code is properly formatted${NC}"
fi
echo ""

# 运行 go vet
echo -e "${YELLOW}11. Running go vet...${NC}"
go vet ./...
echo -e "${GREEN}✓ No issues found by go vet${NC}"
echo ""

# 检查 go.mod
echo -e "${YELLOW}12. Checking go.mod...${NC}"
go mod tidy
if [ -n "$(git diff go.mod go.sum)" ]; then
    echo -e "${YELLOW}⚠ go.mod or go.sum has changes${NC}"
    echo "Run: go mod tidy"
else
    echo -e "${GREEN}✓ go.mod is up to date${NC}"
fi
echo ""

# 总结
echo "========================================="
echo -e "${GREEN}✓ All verifications passed!${NC}"
echo "========================================="
echo ""
echo "Next steps:"
echo "1. Run integration tests with real RabbitMQ"
echo "2. Review RELEASE_CHECKLIST.md"
echo "3. Create git tag and release"
echo ""


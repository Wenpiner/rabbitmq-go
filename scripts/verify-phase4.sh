#!/bin/bash

# 阶段四验证脚本
# 用于验证所有代码编译和基本功能

set -e

echo "========================================="
echo "  阶段四实现验证脚本"
echo "========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 计数器
PASS=0
FAIL=0

# 测试函数
test_build() {
    local name=$1
    local path=$2
    
    echo -n "测试: $name ... "
    if go build -o /tmp/test-$$ "$path" 2>/dev/null; then
        echo -e "${GREEN}✅ 通过${NC}"
        ((PASS++))
        return 0
    else
        echo -e "${RED}❌ 失败${NC}"
        ((FAIL++))
        return 1
    fi
}

echo "1️⃣  编译主包、conf 包和 tracing 包"
echo "-----------------------------------"
test_build "主包" "."
test_build "conf 包" "./conf"
test_build "tracing 包" "./tracing"
echo ""

echo "2️⃣  运行 tracing 包单元测试"
echo "-----------------------------------"
echo -n "测试: tracing 包单元测试 ... "
if go test ./tracing -v > /tmp/test-output-$$ 2>&1; then
    echo -e "${GREEN}✅ 通过${NC}"
    ((PASS++))
    # 显示测试结果
    grep "PASS:" /tmp/test-output-$$ | head -5
else
    echo -e "${RED}❌ 失败${NC}"
    ((FAIL++))
    cat /tmp/test-output-$$
fi
echo ""

echo "3️⃣  编译所有示例（包括阶段四）"
echo "-----------------------------------"
for dir in examples/*/; do
    if [ -f "$dir/main.go" ]; then
        name=$(basename "$dir")
        test_build "示例: $name" "$dir/main.go"
    fi
done
echo ""

echo "4️⃣  检查文档文件"
echo "-----------------------------------"
docs=(
    "PHASE1_IMPLEMENTATION_SUMMARY.md"
    "PHASE1_COMPLETE.md"
    "PHASE2_IMPLEMENTATION_SUMMARY.md"
    "PHASE2_COMPLETE.md"
    "PHASE3_IMPLEMENTATION_SUMMARY.md"
    "PHASE3_COMPLETE.md"
    "PHASE4_IMPLEMENTATION_SUMMARY.md"
    "PHASE4_COMPLETE.md"
    "CONTEXT_IMPLEMENTATION_PROGRESS.md"
    "docs/PHASE1_USAGE.md"
    "docs/PHASE2_USAGE.md"
    "docs/PHASE3_USAGE.md"
    "docs/PHASE4_USAGE.md"
    "examples/README.md"
)

for doc in "${docs[@]}"; do
    echo -n "检查: $doc ... "
    if [ -f "$doc" ]; then
        echo -e "${GREEN}✅ 存在${NC}"
        ((PASS++))
    else
        echo -e "${RED}❌ 缺失${NC}"
        ((FAIL++))
    fi
done
echo ""

echo "5️⃣  检查核心代码修改"
echo "-----------------------------------"
files=(
    "rabbitmq.go"
    "message.go"
    "handler.go"
    "tracing/tracing.go"
    "tracing/tracing_test.go"
)

for file in "${files[@]}"; do
    echo -n "检查: $file ... "
    if [ -f "$file" ]; then
        echo -e "${GREEN}✅ 存在${NC}"
        ((PASS++))
    else
        echo -e "${RED}❌ 缺失${NC}"
        ((FAIL++))
    fi
done
echo ""

echo "6️⃣  检查 tracing 包功能"
echo "-----------------------------------"
echo -n "检查: TraceInfo 结构体 ... "
if grep -q "type TraceInfo struct" tracing/tracing.go; then
    echo -e "${GREEN}✅ 已添加${NC}"
    ((PASS++))
else
    echo -e "${RED}❌ 未找到${NC}"
    ((FAIL++))
fi

echo -n "检查: GenerateTraceID 函数 ... "
if grep -q "func GenerateTraceID" tracing/tracing.go; then
    echo -e "${GREEN}✅ 已添加${NC}"
    ((PASS++))
else
    echo -e "${RED}❌ 未找到${NC}"
    ((FAIL++))
fi

echo -n "检查: InjectToHeaders 函数 ... "
if grep -q "func InjectToHeaders" tracing/tracing.go; then
    echo -e "${GREEN}✅ 已添加${NC}"
    ((PASS++))
else
    echo -e "${RED}❌ 未找到${NC}"
    ((FAIL++))
fi

echo -n "检查: ExtractFromContext 函数 ... "
if grep -q "func ExtractFromContext" tracing/tracing.go; then
    echo -e "${GREEN}✅ 已添加${NC}"
    ((PASS++))
else
    echo -e "${RED}❌ 未找到${NC}"
    ((FAIL++))
fi

echo -n "检查: FormatTraceLog 函数 ... "
if grep -q "func FormatTraceLog" tracing/tracing.go; then
    echo -e "${GREEN}✅ 已添加${NC}"
    ((PASS++))
else
    echo -e "${RED}❌ 未找到${NC}"
    ((FAIL++))
fi

echo -n "检查: SendMessageWithTrace 方法 ... "
if grep -q "func.*SendMessageWithTrace" rabbitmq.go; then
    echo -e "${GREEN}✅ 已添加${NC}"
    ((PASS++))
else
    echo -e "${RED}❌ 未找到${NC}"
    ((FAIL++))
fi

echo -n "检查: handler 集成追踪 ... "
if grep -q "tracing.ExtractFromHeaders" handler.go; then
    echo -e "${GREEN}✅ 已集成${NC}"
    ((PASS++))
else
    echo -e "${RED}❌ 未集成${NC}"
    ((FAIL++))
fi
echo ""

echo "========================================="
echo "  验证结果"
echo "========================================="
echo ""
echo -e "通过: ${GREEN}$PASS${NC}"
echo -e "失败: ${RED}$FAIL${NC}"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}🎉 所有验证通过！阶段四实现完成。${NC}"
    exit 0
else
    echo -e "${RED}⚠️  有 $FAIL 项验证失败，请检查。${NC}"
    exit 1
fi


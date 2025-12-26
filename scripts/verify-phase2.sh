#!/bin/bash

# 阶段二验证脚本
# 用于验证所有代码编译和基本功能

set -e

echo "========================================="
echo "  阶段二实现验证脚本"
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

echo "1️⃣  编译主包和 conf 包"
echo "-----------------------------------"
test_build "主包" "."
test_build "conf 包" "./conf"
echo ""

echo "2️⃣  编译所有示例（包括阶段二）"
echo "-----------------------------------"
for dir in examples/*/; do
    if [ -f "$dir/main.go" ]; then
        name=$(basename "$dir")
        test_build "示例: $name" "$dir/main.go"
    fi
done
echo ""

echo "3️⃣  检查文档文件"
echo "-----------------------------------"
docs=(
    "PHASE1_IMPLEMENTATION_SUMMARY.md"
    "PHASE1_COMPLETE.md"
    "PHASE2_IMPLEMENTATION_SUMMARY.md"
    "PHASE2_COMPLETE.md"
    "docs/PHASE1_USAGE.md"
    "docs/PHASE2_USAGE.md"
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

echo "4️⃣  检查核心代码修改"
echo "-----------------------------------"
files=(
    "message.go"
    "handler.go"
    "rabbitmq.go"
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

echo "5️⃣  检查函数签名"
echo "-----------------------------------"
echo -n "检查: SendDelayMsg 支持 context ... "
if grep -q "func.*SendDelayMsg.*ctx context.Context" message.go; then
    echo -e "${GREEN}✅ 已添加${NC}"
    ((PASS++))
else
    echo -e "${RED}❌ 未找到${NC}"
    ((FAIL++))
fi

echo -n "检查: SendDelayMsgByArgs 支持 context ... "
if grep -q "func.*SendDelayMsgByArgs.*ctx context.Context" message.go; then
    echo -e "${GREEN}✅ 已添加${NC}"
    ((PASS++))
else
    echo -e "${RED}❌ 未找到${NC}"
    ((FAIL++))
fi

echo -n "检查: SendDelayMsgByKey 支持 context ... "
if grep -q "func.*SendDelayMsgByKey.*ctx context.Context" rabbitmq.go; then
    echo -e "${GREEN}✅ 已添加${NC}"
    ((PASS++))
else
    echo -e "${RED}❌ 未找到${NC}"
    ((FAIL++))
fi

echo -n "检查: 兼容性函数 SendDelayMsgCompat ... "
if grep -q "func.*SendDelayMsgCompat" message.go; then
    echo -e "${GREEN}✅ 已添加${NC}"
    ((PASS++))
else
    echo -e "${RED}❌ 未找到${NC}"
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
    echo -e "${GREEN}🎉 所有验证通过！阶段二实现完成。${NC}"
    exit 0
else
    echo -e "${RED}⚠️  有 $FAIL 项验证失败，请检查。${NC}"
    exit 1
fi


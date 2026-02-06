#!/bin/bash

# 验证稳定性测试环境设置

# 不使用 set -e，因为我们需要收集所有错误

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}  验证稳定性测试环境设置${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

PASS=0
FAIL=0

check_file() {
    local file=$1
    local desc=$2
    
    if [ -f "$file" ]; then
        echo -e "${GREEN}✅${NC} $desc: $file"
        ((PASS++))
    else
        echo -e "${RED}❌${NC} $desc: $file (缺失)"
        ((FAIL++))
    fi
}

check_dir() {
    local dir=$1
    local desc=$2
    
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✅${NC} $desc: $dir"
        ((PASS++))
    else
        echo -e "${RED}❌${NC} $desc: $dir (缺失)"
        ((FAIL++))
    fi
}

echo -e "${YELLOW}1. 检查 Docker 配置文件${NC}"
check_file "docker-compose.yml" "Docker Compose 配置"
check_file "test/docker/rabbitmq/rabbitmq.conf" "RabbitMQ 配置"
check_file "test/docker/rabbitmq/enabled_plugins" "RabbitMQ 插件"
check_file "test/docker/prometheus/prometheus.yml" "Prometheus 配置"
check_file "test/docker/grafana/provisioning/datasources/prometheus.yml" "Grafana 数据源"
check_file "test/docker/grafana/provisioning/dashboards/dashboard.yml" "Grafana 仪表板配置"
check_file "test/docker/grafana/dashboards/rabbitmq-stability.json" "Grafana 仪表板"
echo ""

echo -e "${YELLOW}2. 检查 Dockerfile${NC}"
check_file "test/docker/stability/Dockerfile.long-run" "长时间测试 Dockerfile"
check_file "test/docker/stability/Dockerfile.high-concurrency" "高并发测试 Dockerfile"
check_file "test/docker/stability/Dockerfile.network-chaos" "网络故障测试 Dockerfile"
check_file "test/docker/stability/Dockerfile.memory-leak" "内存泄漏测试 Dockerfile"
echo ""

echo -e "${YELLOW}3. 检查测试代码${NC}"
check_file "test/stability/common/metrics.go" "指标收集库"
check_file "test/stability/common/config.go" "配置加载库"
check_file "test/stability/long-run/main.go" "长时间测试程序"
check_file "test/stability/high-concurrency/main.go" "高并发测试程序"
check_file "test/stability/network-chaos/main.go" "网络故障测试程序"
check_file "test/stability/memory-leak/main.go" "内存泄漏测试程序"
echo ""

echo -e "${YELLOW}4. 检查脚本和工具${NC}"
check_file "scripts/run-stability-tests.sh" "测试运行脚本"
check_file "Makefile" "Makefile"
check_file ".github/workflows/stability-test.yml" "GitHub Actions 配置"
echo ""

echo -e "${YELLOW}5. 检查文档${NC}"
check_file "test/STABILITY_TEST_README.md" "完整测试指南"
check_file "test/STABILITY_QUICKSTART.md" "快速开始指南"
check_file "test/stability/README.md" "测试代码说明"
check_file "STABILITY_TEST_SUMMARY.md" "测试方案总结"
check_file "STABILITY_TEST_EXAMPLE.md" "实战示例"
echo ""

echo -e "${YELLOW}6. 检查脚本权限${NC}"
if [ -x "scripts/run-stability-tests.sh" ]; then
    echo -e "${GREEN}✅${NC} 测试脚本可执行"
    ((PASS++))
else
    echo -e "${RED}❌${NC} 测试脚本不可执行"
    ((FAIL++))
fi
echo ""

echo -e "${YELLOW}7. 验证 Go 代码编译${NC}"
if go build ./test/stability/common 2>/dev/null; then
    echo -e "${GREEN}✅${NC} common 包编译成功"
    ((PASS++))
else
    echo -e "${RED}❌${NC} common 包编译失败"
    ((FAIL++))
fi

for test_dir in long-run high-concurrency network-chaos memory-leak; do
    if go build ./test/stability/$test_dir 2>/dev/null; then
        echo -e "${GREEN}✅${NC} $test_dir 编译成功"
        ((PASS++))
    else
        echo -e "${RED}❌${NC} $test_dir 编译失败"
        ((FAIL++))
    fi
done
echo ""

echo -e "${YELLOW}8. 检查 Docker 环境${NC}"
if command -v docker &> /dev/null; then
    echo -e "${GREEN}✅${NC} Docker 已安装"
    ((PASS++))
else
    echo -e "${RED}❌${NC} Docker 未安装"
    ((FAIL++))
fi

if command -v docker-compose &> /dev/null; then
    echo -e "${GREEN}✅${NC} Docker Compose 已安装"
    ((PASS++))
else
    echo -e "${RED}❌${NC} Docker Compose 未安装"
    ((FAIL++))
fi
echo ""

echo "========================================="
echo -e "总计: ${GREEN}$PASS 通过${NC}, ${RED}$FAIL 失败${NC}"
echo "========================================="
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}✅ 所有检查通过！稳定性测试环境已就绪。${NC}"
    echo ""
    echo "下一步:"
    echo "  1. 运行快速测试: make stability-long-run"
    echo "  2. 查看文档: cat test/STABILITY_QUICKSTART.md"
    echo "  3. 打开监控: make stability-monitor"
    echo ""
    exit 0
else
    echo -e "${RED}❌ 有 $FAIL 项检查失败，请修复后重试。${NC}"
    exit 1
fi


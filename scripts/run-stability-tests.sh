#!/bin/bash

# 稳定性测试运行脚本

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}  RabbitMQ-Go 稳定性测试套件${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# 显示帮助信息
show_help() {
    echo "用法: $0 [选项] [测试类型]"
    echo ""
    echo "测试类型:"
    echo "  all              - 运行所有测试"
    echo "  long-run         - 长时间稳定性测试 (24小时)"
    echo "  high-concurrency - 高并发压力测试"
    echo "  network-chaos    - 网络故障恢复测试"
    echo "  memory-leak      - 内存泄漏检测测试"
    echo ""
    echo "选项:"
    echo "  -h, --help       - 显示帮助信息"
    echo "  -d, --down       - 停止并清理所有容器"
    echo "  -l, --logs       - 查看测试日志"
    echo "  -s, --status     - 查看测试状态"
    echo "  -m, --monitor    - 打开监控面板"
    echo ""
    echo "示例:"
    echo "  $0 long-run              # 运行长时间测试"
    echo "  $0 high-concurrency      # 运行高并发测试"
    echo "  $0 -l long-run           # 查看长时间测试日志"
    echo "  $0 -d                    # 停止所有测试"
}

# 检查 Docker
check_docker() {
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}❌ Docker 未安装${NC}"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        echo -e "${RED}❌ Docker Compose 未安装${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ Docker 环境检查通过${NC}"
}

# 启动基础服务
start_infrastructure() {
    echo -e "${YELLOW}启动基础服务 (RabbitMQ, Prometheus, Grafana)...${NC}"
    docker-compose up -d rabbitmq prometheus grafana
    
    echo -e "${YELLOW}等待 RabbitMQ 就绪...${NC}"
    sleep 10
    
    # 检查 RabbitMQ 健康状态
    for i in {1..30}; do
        if docker-compose exec -T rabbitmq rabbitmq-diagnostics ping &> /dev/null; then
            echo -e "${GREEN}✅ RabbitMQ 已就绪${NC}"
            break
        fi
        echo -n "."
        sleep 2
    done
    
    echo ""
    echo -e "${GREEN}✅ 基础服务已启动${NC}"
    echo ""
    echo -e "${BLUE}访问地址:${NC}"
    echo -e "  RabbitMQ 管理界面: ${GREEN}http://localhost:15672${NC} (guest/guest)"
    echo -e "  Prometheus:       ${GREEN}http://localhost:9090${NC}"
    echo -e "  Grafana:          ${GREEN}http://localhost:3000${NC} (admin/admin)"
    echo ""
}

# 运行长时间测试
run_long_run() {
    echo -e "${YELLOW}启动长时间稳定性测试...${NC}"
    docker-compose up -d stability-long-run
    echo -e "${GREEN}✅ 长时间测试已启动${NC}"
    echo -e "查看日志: ${BLUE}docker-compose logs -f stability-long-run${NC}"
}

# 运行高并发测试
run_high_concurrency() {
    echo -e "${YELLOW}启动高并发压力测试...${NC}"
    docker-compose --profile high-concurrency up -d stability-high-concurrency
    echo -e "${GREEN}✅ 高并发测试已启动${NC}"
    echo -e "查看日志: ${BLUE}docker-compose logs -f stability-high-concurrency${NC}"
}

# 运行网络故障测试
run_network_chaos() {
    echo -e "${YELLOW}启动网络故障恢复测试...${NC}"
    docker-compose --profile chaos up -d stability-network-chaos
    echo -e "${GREEN}✅ 网络故障测试已启动${NC}"
    echo -e "查看日志: ${BLUE}docker-compose logs -f stability-network-chaos${NC}"
}

# 运行内存泄漏测试
run_memory_leak() {
    echo -e "${YELLOW}启动内存泄漏检测测试...${NC}"
    docker-compose --profile memory-leak up -d stability-memory-leak
    echo -e "${GREEN}✅ 内存泄漏测试已启动${NC}"
    echo -e "查看日志: ${BLUE}docker-compose logs -f stability-memory-leak${NC}"
    echo -e "pprof 地址: ${BLUE}http://localhost:6060/debug/pprof/${NC}"
}

# 查看状态
show_status() {
    echo -e "${BLUE}=== 容器状态 ===${NC}"
    docker-compose ps
    echo ""
    
    echo -e "${BLUE}=== 资源使用 ===${NC}"
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" \
        $(docker-compose ps -q 2>/dev/null)
}

# 查看日志
show_logs() {
    local service=$1
    if [ -z "$service" ]; then
        docker-compose logs -f
    else
        docker-compose logs -f "stability-$service"
    fi
}

# 停止并清理
cleanup() {
    echo -e "${YELLOW}停止所有测试容器...${NC}"
    docker-compose --profile high-concurrency --profile chaos --profile memory-leak down -v
    echo -e "${GREEN}✅ 清理完成${NC}"
}

# 打开监控面板
open_monitor() {
    echo -e "${BLUE}打开监控面板...${NC}"
    
    if command -v open &> /dev/null; then
        open http://localhost:3000
    elif command -v xdg-open &> /dev/null; then
        xdg-open http://localhost:3000
    else
        echo -e "${YELLOW}请手动打开: http://localhost:3000${NC}"
    fi
}

# 主逻辑
main() {
    case "$1" in
        -h|--help)
            show_help
            exit 0
            ;;
        -d|--down)
            cleanup
            exit 0
            ;;
        -s|--status)
            show_status
            exit 0
            ;;
        -l|--logs)
            show_logs "$2"
            exit 0
            ;;
        -m|--monitor)
            open_monitor
            exit 0
            ;;
        "")
            show_help
            exit 0
            ;;
    esac
    
    check_docker
    start_infrastructure
    
    case "$1" in
        all)
            run_long_run
            run_high_concurrency
            run_network_chaos
            run_memory_leak
            ;;
        long-run)
            run_long_run
            ;;
        high-concurrency)
            run_high_concurrency
            ;;
        network-chaos)
            run_network_chaos
            ;;
        memory-leak)
            run_memory_leak
            ;;
        *)
            echo -e "${RED}❌ 未知的测试类型: $1${NC}"
            show_help
            exit 1
            ;;
    esac
    
    echo ""
    echo -e "${GREEN}=========================================${NC}"
    echo -e "${GREEN}  测试已启动${NC}"
    echo -e "${GREEN}=========================================${NC}"
    echo ""
    echo -e "查看状态: ${BLUE}$0 -s${NC}"
    echo -e "查看日志: ${BLUE}$0 -l${NC}"
    echo -e "打开监控: ${BLUE}$0 -m${NC}"
    echo -e "停止测试: ${BLUE}$0 -d${NC}"
    echo ""
}

main "$@"


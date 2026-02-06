.PHONY: help test test-unit test-integration stability-test stability-long-run stability-high-concurrency stability-network-chaos stability-memory-leak stability-status stability-logs stability-down stability-monitor build clean

# 默认目标
help:
	@echo "RabbitMQ-Go 项目命令"
	@echo ""
	@echo "开发命令:"
	@echo "  make build                    - 编译项目"
	@echo "  make test                     - 运行所有测试"
	@echo "  make test-unit                - 运行单元测试"
	@echo "  make test-integration         - 运行集成测试"
	@echo "  make clean                    - 清理构建产物"
	@echo ""
	@echo "稳定性测试:"
	@echo "  make stability-long-run       - 运行长时间稳定性测试 (24h)"
	@echo "  make stability-high-concurrenicy - 运行高并发压力测试 (1h)"
	@echo "  make stability-network-chaos  - 运行网络故障恢复测试 (2h)"
	@echo "  make stability-memory-leak    - 运行内存泄漏检测测试 (12h)"
	@echo "  make stability-all            - 运行所有稳定性测试"
	@echo ""
	@echo "稳定性测试管理:"
	@echo "  make stability-status         - 查看测试状态"
	@echo "  make stability-logs           - 查看测试日志"
	@echo "  make stability-monitor        - 打开监控面板"
	@echo "  make stability-down           - 停止并清理测试环境"
	@echo ""

# 编译
build:
	@echo "编译项目..."
	@go build ./...
	@echo "✅ 编译完成"

# 运行所有测试
test: test-unit test-integration

# 单元测试
test-unit:
	@echo "运行单元测试..."
	@go test -v -short ./...

# 集成测试
test-integration:
	@echo "运行集成测试..."
	@go test -v -run TestIntegration

# 清理
clean:
	@echo "清理构建产物..."
	@go clean -cache -testcache
	@find . -name "*.test" -delete
	@echo "✅ 清理完成"

# 稳定性测试 - 长时间运行
stability-long-run:
	@./scripts/run-stability-tests.sh long-run

# 稳定性测试 - 高并发
stability-high-concurrency:
	@./scripts/run-stability-tests.sh high-concurrency

# 稳定性测试 - 网络故障
stability-network-chaos:
	@./scripts/run-stability-tests.sh network-chaos

# 稳定性测试 - 内存泄漏
stability-memory-leak:
	@./scripts/run-stability-tests.sh memory-leak

# 稳定性测试 - 全部
stability-all:
	@./scripts/run-stability-tests.sh all

# 查看稳定性测试状态
stability-status:
	@./scripts/run-stability-tests.sh -s

# 查看稳定性测试日志
stability-logs:
	@./scripts/run-stability-tests.sh -l

# 打开监控面板
stability-monitor:
	@./scripts/run-stability-tests.sh -m

# 停止稳定性测试
stability-down:
	@./scripts/run-stability-tests.sh -d

# Docker 相关
docker-build:
	@echo "构建 Docker 镜像..."
	@docker-compose build

docker-up:
	@echo "启动 Docker 服务..."
	@docker-compose up -d

docker-down:
	@echo "停止 Docker 服务..."
	@docker-compose down

docker-logs:
	@docker-compose logs -f

# 代码质量检查
lint:
	@echo "运行代码检查..."
	@golangci-lint run ./...

fmt:
	@echo "格式化代码..."
	@go fmt ./...

vet:
	@echo "运行 go vet..."
	@go vet ./...

# 依赖管理
deps:
	@echo "下载依赖..."
	@go mod download

deps-tidy:
	@echo "整理依赖..."
	@go mod tidy

deps-verify:
	@echo "验证依赖..."
	@go mod verify

# 覆盖率
coverage:
	@echo "生成测试覆盖率报告..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告已生成: coverage.html"

# 基准测试
bench:
	@echo "运行基准测试..."
	@go test -bench=. -benchmem ./...

# 示例运行
example-basic:
	@cd examples/01-basic && go run main.go

example-batch:
	@cd examples/02-batch-publish && go run main.go

example-tracing:
	@cd examples/03-tracing && go run main.go

# 完整验证
verify: clean build test lint
	@echo "✅ 所有验证通过"


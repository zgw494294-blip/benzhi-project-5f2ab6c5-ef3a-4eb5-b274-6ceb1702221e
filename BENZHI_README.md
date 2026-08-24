# BENZHI_README

基于 Go 实现的城市树木健康巡检与修复验收台 Web 项目，一款后端服务，已完整实现城市树木健康巡检与修复验收台。规定的测试与自检命令均通过，桌面和窄屏页面已完成截图核对；当前工作台运行于 http://127.0.0.1:19083。

## 项目说明
- 项目：benzhi-project-5f2ab6c5-ef3a-4eb5-b274-6ceb1702221e
- 项目用途：已完整实现城市树木健康巡检与修复验收台。规定的测试与自检命令均通过，桌面和窄屏页面已完成截图核对；当前工作台运行于 http://127.0.0.1:19083。
- Go 工具链：`golang:1.24`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run . -selfcheck -addr=127.0.0.1:19082
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-5f2ab6c5-ef3a-4eb5-b274-6ceb1702221e-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-5f2ab6c5-ef3a-4eb5-b274-6ceb1702221e-arm64 linux/arm64
docker run -it benzhi-project-5f2ab6c5-ef3a-4eb5-b274-6ceb1702221e-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run . -selfcheck -addr=127.0.0.1:19082`

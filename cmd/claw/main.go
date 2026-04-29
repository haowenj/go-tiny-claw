// cmd/claw/main.go
package main

import (
	"context"
	"log"
	"os"

	"github.com/haowenj/go-tiny-claw/internal/engine"
	"github.com/haowenj/go-tiny-claw/internal/logutil"
	"github.com/haowenj/go-tiny-claw/internal/provider"
	"github.com/haowenj/go-tiny-claw/internal/schema"
	"github.com/haowenj/go-tiny-claw/internal/tools"
)

// 伪造的工具注册表 (用于测试 Provider 的工具提取能力)
type mockRegistry struct{}

func (m *mockRegistry) GetAvailableTools() []schema.ToolDefinition {
	return []schema.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "获取指定城市的当前天气情况。",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"city": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"city"},
			},
		},
	}
}

func (m *mockRegistry) Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult {
	log.Printf("  -> [Mock 工具执行] 获取 %s 的天气中...\n", call.Name)
	return schema.ToolResult{
		ToolCallID: call.ID,
		Output:     "API 返回：今天是晴天，气温 25 度。",
		IsError:    false,
	}
}

func main() {
	logutil.Init()

	if os.Getenv("API_KEY") == "" {
		log.Fatal("请先导出 API_KEY 环境变量")
	}

	workDir, _ := os.Getwd()

	// 1. 初始化真实的 Provider大脑
	llmProvider := provider.NewOpenAIProvider("qwen3-max")

	// 2. 注入伪造的工具注册表
	registry := tools.NewRegistry()

	// 3. 将真实的 ReadFile 工具挂载到注册表中
	readFileTool := tools.NewReadFileTool(workDir)
	registry.Register(readFileTool)

	// 4. 实例化核心引擎，由于任务简单，我们关闭思考阶段 (EnableThinking = false) 以加快速度
	eng := engine.NewAgentEngine(llmProvider, registry, workDir, false)

	// 设定测试任务
	prompt := "请调用工具读取一下当前工作区目录下 hello.txt 文件的内容，并用一句话向我总结它说了什么。"

	err := eng.Run(context.Background(), prompt)
	if err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}

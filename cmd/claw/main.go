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

	// 2. 初始化工具工厂
	registry := tools.NewRegistry()

	// 3. 实例化读取文件的工具并注册到工具工厂中
	registry.Register(tools.NewReadFileTool(workDir))
	registry.Register(tools.NewWriteFileTool(workDir))
	registry.Register(tools.NewBashTool(workDir))

	// 4. 实例化核心引擎，由于任务简单，我们关闭思考阶段 (EnableThinking = false) 以加快速度
	eng := engine.NewAgentEngine(llmProvider, registry, workDir, false)

	// 设定测试任务
	prompt := `
	请帮我执行以下操作：
	1、用 bash 查看一下我当前电脑的 Go 版本。
	2、帮我写一个简单的 helloworld.go 文件，输出 "Hello, go-tiny-claw!"。
	3、将代码输出到出来
	4、用 bash 编译并运行这个 go 文件，确认它能正常工作。
	`

	err := eng.Run(context.Background(), prompt)
	if err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}

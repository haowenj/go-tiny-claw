package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/haowenj/go-tiny-claw/internal/schema"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

type CodexProvider struct {
	client openai.Client
	model  string
}

func NewCodexProvider(model string) *CodexProvider {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		panic("请设置 OPENAI_API_KEY 环境变量")
	}

	baseURL := "https://api.openai.com/v1/"

	return &CodexProvider{
		client: openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL)),
		model:  model,
	}
}

func (p *CodexProvider) Generate(ctx context.Context, msgs []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
	params := buildCodexResponseParams(p.model, msgs, availableTools)

	resp, err := p.client.Responses.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("Codex/Responses API 请求失败: %w", err)
	}

	return parseCodexResponse(resp)
}

func buildCodexResponseParams(model string, msgs []schema.Message, availableTools []schema.ToolDefinition) responses.ResponseNewParams {
	instructions, input := buildCodexInput(msgs)

	params := responses.ResponseNewParams{
		Model:             shared.ResponsesModel(model),
		Input:             responses.ResponseNewParamsInputUnion{OfInputItemList: input},
		MaxOutputTokens:   openai.Int(4096),
		ParallelToolCalls: openai.Bool(true),
	}

	if instructions != "" {
		params.Instructions = openai.String(instructions)
	}

	if len(availableTools) > 0 {
		params.Tools = buildCodexTools(availableTools)
	}

	return params
}

func buildCodexInput(msgs []schema.Message) (string, responses.ResponseInputParam) {
	var instructions string
	var input responses.ResponseInputParam

	for idx, msg := range msgs {
		switch msg.Role {
		case schema.RoleSystem:
			instructions = msg.Content

		case schema.RoleUser:
			if msg.ToolCallID != "" {
				input = append(input, responses.ResponseInputItemParamOfFunctionCallOutput(msg.ToolCallID, msg.Content))
				continue
			}

			input = append(input, responses.ResponseInputItemParamOfInputMessage(
				responses.ResponseInputMessageContentListParam{
					responses.ResponseInputContentParamOfInputText(msg.Content),
				},
				string(schema.RoleUser),
			))

		case schema.RoleAssistant:
			if msg.Content != "" {
				input = append(input, responses.ResponseInputItemParamOfOutputMessage(
					[]responses.ResponseOutputMessageContentUnionParam{
						{OfOutputText: &responses.ResponseOutputTextParam{Text: msg.Content}},
					},
					fmt.Sprintf("assistant_%d", idx),
					responses.ResponseOutputMessageStatusCompleted,
				))

				if len(msg.ToolCalls) > 0 {
					input[len(input)-1].OfOutputMessage.Phase = responses.ResponseOutputMessagePhaseCommentary
				} else {
					input[len(input)-1].OfOutputMessage.Phase = responses.ResponseOutputMessagePhaseFinalAnswer
				}
			}

			for _, tc := range msg.ToolCalls {
				input = append(input, responses.ResponseInputItemParamOfFunctionCall(string(tc.Arguments), tc.ID, tc.Name))
			}
		}
	}

	return instructions, input
}

func buildCodexTools(availableTools []schema.ToolDefinition) []responses.ToolUnionParam {
	tools := make([]responses.ToolUnionParam, 0, len(availableTools))

	for _, toolDef := range availableTools {
		params := map[string]any{}
		if m, ok := toolDef.InputSchema.(map[string]any); ok {
			params = m
		} else {
			b, _ := json.Marshal(toolDef.InputSchema)
			_ = json.Unmarshal(b, &params)
		}

		tools = append(tools, responses.ToolParamOfFunction(toolDef.Name, params, false))
		tools[len(tools)-1].OfFunction.Description = openai.String(toolDef.Description)
	}

	return tools
}

func parseCodexResponse(resp *responses.Response) (*schema.Message, error) {
	if resp == nil {
		return nil, fmt.Errorf("Responses API 返回为空")
	}

	if resp.Error.Code != "" || resp.Status == responses.ResponseStatusFailed {
		return nil, fmt.Errorf("Responses API 返回失败: %s", resp.Error.Message)
	}

	resultMsg := &schema.Message{Role: schema.RoleAssistant}

	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			for _, content := range item.Content {
				switch content.Type {
				case "output_text":
					resultMsg.Content += content.Text
				case "refusal":
					resultMsg.Content += content.Refusal
				}
			}

		case "function_call":
			resultMsg.ToolCalls = append(resultMsg.ToolCalls, schema.ToolCall{
				ID:        item.CallID,
				Name:      item.Name,
				Arguments: []byte(item.Arguments.OfString),
			})
		}
	}

	return resultMsg, nil
}

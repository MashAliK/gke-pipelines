package agent

import (
	"fmt"
	"context"
	_ "embed"
	"strings"

	"github.com/MashAliK/gke-pipelines/internal/client"
	"github.com/MashAliK/gke-pipelines/internal/tool"
	"github.com/GoogleCloudPlatform/kubectl-ai/gollm"
)

//go:embed system_prompt.txt
var SystemPrompt string

type Agent struct {
	LLM gollm.Client

	Model string

	Provider string

	Chat gollm.Chat

	KubectlAIClient *client.KubectlClient
}

func (a* Agent) Init(ctx context.Context) error {
	system_prompt := a.getPrompt()

	agentTools := []*gollm.FunctionDefinition{tool.NewKubectlAITool(), tool.NewCommandTool(), tool.NewMessageTool()}

    a.Chat = a.LLM.StartChat(system_prompt, a.Model)
    
	a.Chat.SetFunctionDefinitions(agentTools)

	return nil
}

func (a* Agent) SendMessage(ctx context.Context, message string) (string, error) {
	response, err := a.Chat.Send(ctx, message)
	if err != nil {
		return "", err
	}
	
	var messages strings.Builder
	for {
		candidates := response.Candidates()
		newMessageSent := false
		
		for i := 0; i < len(candidates); i++ {
			candidate := candidates[i]
			for _, part := range candidate.Parts() {
				if functionCalls, ok := part.AsFunctionCalls(); ok {
					for _, call := range functionCalls {
						var result map[string]any
						
						switch call.Name {
						case "kubectl-ai":
							queryResult := a.KubectlAIClient.Query(ctx, call.Arguments["intent"].(string))
							result = map[string]any{"response": queryResult}
							
						case "exec-command":
							cmdResult, _ := tool.ExecuteCommand(ctx, call.Arguments["command"].(string))
							messages.WriteString(fmt.Sprintf("Result: %s", cmdResult))
							result = map[string]any{"response": cmdResult}
							
						case "message-agent":
							messages.WriteString(fmt.Sprintf("%s\n", call.Arguments["response"].(string)))
							result = map[string]any{"response": "Message sent to agent!"}
						}
						
						newResponse, err := a.Chat.Send(ctx, gollm.FunctionCallResult{
							ID:     call.ID,
							Name:   call.Name,
							Result: result,
						})
						if err != nil {
							return "", err
						}
						
						candidates = append(candidates, newResponse.Candidates()...)
						response = newResponse
						newMessageSent = true
					}
				}
			}
		}

		if !newMessageSent {
			break
		}
	}
	return messages.String(), nil
}

func (a* Agent) getPrompt() (string) {
	return SystemPrompt
}
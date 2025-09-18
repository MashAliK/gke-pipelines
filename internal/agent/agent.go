package agent

import (
	"os"
	"fmt"
	"context"

	"github.com/MashAliK/gke-pipelines/internal/tool"
	"github.com/GoogleCloudPlatform/kubectl-ai/gollm"
	exttools "github.com/GoogleCloudPlatform/kubectl-ai/pkg/tools"
)

type Agent struct {
	LLM gollm.Client

	Model string

	Provider string

	Chat gollm.Chat
}

func (a* Agent) Init(ctx context.Context) error {
	system_prompt, err := a.getPrompt()
	if err != nil {
		return err
	}

	bashtool := &exttools.BashTool{}

	agentTools := []*gollm.FunctionDefinition{tool.NewKubectlAITool(), bashtool.FunctionDefinition()}

    a.Chat = a.LLM.StartChat(system_prompt, a.Model)
    
	a.Chat.SetFunctionDefinitions(agentTools)

	return nil
}

func (a* Agent) SendMessage(ctx context.Context, message string) error {
	response, err := a.Chat.Send(ctx, message)
	if err != nil {
		return err
	}

	for _, candidate := range response.Candidates() {
		for _, part := range candidate.Parts() {
			if text, ok := part.AsText(); ok {
				fmt.Print("Text response: ")
				fmt.Println(text)
			}

			if functionCalls, ok := part.AsFunctionCalls(); ok {
				for _, call := range functionCalls {
					fmt.Printf("Function call: %s with args %v\n", call.Name, call.Arguments)
				}
			}
		}
	}
	return nil
}

func (a* Agent) getPrompt() (string, error) {
	content, err := os.ReadFile("./internal/agent/system_prompt.txt")
	if err != nil {
		return "", err
	}

	text := string(content)
	return text, err
}
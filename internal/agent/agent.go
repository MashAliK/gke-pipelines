package agent

import (
	"os"
	"fmt"
	"context"

	"github.com/MashAliK/gke-pipelines/internal/tool"
	"github.com/MashAliK/gke-pipelines/internal/client"
	"github.com/GoogleCloudPlatform/kubectl-ai/gollm"
)

type Agent struct {
	LLM gollm.Client

	Model string

	Provider string

	Chat gollm.Chat

	KubectlAIClient *client.KubectlClient
}

func (a* Agent) Init(ctx context.Context) error {
	system_prompt, err := a.getPrompt()
	if err != nil {
		return err
	}

	agentTools := []*gollm.FunctionDefinition{tool.NewKubectlAITool()}

    a.Chat = a.LLM.StartChat(system_prompt, a.Model)
    
	a.Chat.SetFunctionDefinitions(agentTools)

	return nil
}

func (a* Agent) SendMessage(ctx context.Context, message string) error {
	response, err := a.Chat.Send(ctx, message)
	if err != nil {
		return err
	}
	
	for newMessageSent := true; newMessageSent; {
		newMessageSent = false
		for _, candidate := range response.Candidates() {
			for _, part := range candidate.Parts() {
				if text, ok := part.AsText(); ok {
					fmt.Print("Text response: ")
					fmt.Println(text)
				}

				if functionCalls, ok := part.AsFunctionCalls(); ok {
					for _, call := range functionCalls {
						if call.Name == "kubectl-ai" {
							fmt.Printf("Making tool call: %s\n", call.Arguments["Intent"].(string))
							result := a.KubectlAIClient.Query(ctx, call.Arguments["Intent"].(string))
							response, err = a.Chat.Send(ctx, gollm.FunctionCallResult{
								ID:     call.ID,
								Name:   call.Name,
								Result: map[string]any{"response": result},
							})
							if err != nil {
								return err
							}
							newMessageSent = true
						} else {
							fmt.Printf("Function call: %s with args %v\n", call.Name, call.Arguments)
						}
					}
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
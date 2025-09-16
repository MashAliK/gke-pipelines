package agent

import (
	"fmt"
	"context"

	"github.com/GoogleCloudPlatform/kubectl-ai/gollm"
)

type Agent struct {
	LLM gollm.Client

	Model string

	Provider string

	Chat gollm.Chat

	Tools []*gollm.FunctionDefinition
}

func (a* Agent) Init(ctx context.Context) error {
	const system_prompt = "Assist the user by calling the relevant tools."

    a.Chat = a.LLM.StartChat(system_prompt, a.Model)
    
	a.Chat.SetFunctionDefinitions(a.Tools)

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
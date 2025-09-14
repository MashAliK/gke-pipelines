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
}

func (s* Agent) Init(ctx context.Context) error {
	const system_prompt = "Respond in pig latin."

    // Start a chat conversation
    chat := s.LLM.StartChat(system_prompt, s.Model)
    
    // Send a message
    response, err := chat.Send(ctx, "Hello, how are you?")
    if err != nil {
        return err
	}

	// Print the response
    for _, candidate := range response.Candidates() {
        fmt.Println(candidate.String())
    }

	return nil
}

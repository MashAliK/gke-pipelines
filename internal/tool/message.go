package tool

import (
	"github.com/GoogleCloudPlatform/kubectl-ai/gollm"
)

func NewMessageTool() *gollm.FunctionDefinition {
	return &gollm.FunctionDefinition{
		Name:        "message-agent",
		Description: "Send a message to the agent assisting the user.",
		Parameters: &gollm.Schema{
			Type: gollm.TypeObject,
			Properties: map[string]*gollm.Schema{
				"response": {
					Type:        gollm.TypeString,
					Description: "Message to be sent to the agent. This could be a response to their query or a follow-up question. Include all relevant details if the agent asked for them (i.e. logs).",
				},
			},
			Required: []string{"response"},
		},
	}
}
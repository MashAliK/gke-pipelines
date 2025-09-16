package tool

import (
	"github.com/GoogleCloudPlatform/kubectl-ai/gollm"
)

func NewKubectlAITool() *gollm.FunctionDefinition {
	return  &gollm.FunctionDefinition{
		Name: "kubectl-ai",
		Description: "Assist with kubernetes-related tasks.",
		Parameters: &gollm.Schema{
			Type: gollm.TypeObject,
			Properties: map[string]*gollm.Schema{
				"Intent": {
					Type:        gollm.TypeString,
					Description: "The task for the kubernetes agent to complete.",
				},
        	},
		},
	}
}


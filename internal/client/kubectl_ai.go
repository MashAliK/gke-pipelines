package client

import (
	"fmt"
    "context"
	"strings"

    "github.com/MashAliK/gke-pipelines/internal/config"
	"github.com/GoogleCloudPlatform/kubectl-ai/gollm"
	"github.com/GoogleCloudPlatform/kubectl-ai/pkg/agent"
	"github.com/GoogleCloudPlatform/kubectl-ai/pkg/api"
	"github.com/GoogleCloudPlatform/kubectl-ai/pkg/tools"
	"github.com/GoogleCloudPlatform/kubectl-ai/pkg/journal"
	"github.com/GoogleCloudPlatform/kubectl-ai/pkg/sessions"
)

type KubectlClient struct {
	agent		*agent.Agent
	messages	chan string
}

func NewKubectlClient(ctx context.Context, llmClient *gollm.Client) (*KubectlClient, error) {
	var config config.KubectlAIOptions
	config.InitDefaults()

	var recorder journal.Recorder
	// use in-memory session
	chatStore := sessions.NewInMemoryChatStore()

	k8sAgent := &agent.Agent{
		Model:              config.ModelID,
		Provider:           config.ProviderID,
		Kubeconfig:         config.KubeConfigPath,
		LLM:                *llmClient,
		MaxIterations:      config.MaxIterations,
		PromptTemplateFile: config.PromptTemplateFilePath,
		ExtraPromptPaths:   config.ExtraPromptPaths,
		Tools:              tools.Default(),
		Recorder:           recorder,
		RemoveWorkDir:      config.RemoveWorkDir,
		SkipPermissions:    config.SkipPermissions,
		EnableToolUseShim:  config.EnableToolUseShim,
		MCPClientEnabled:   config.MCPClient,
		RunOnce:            config.Quiet,
		InitialQuery:       "",
		ChatMessageStore:   chatStore,
	}

	err := k8sAgent.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting k8s agent: %w", err)
	}
	defer k8sAgent.Close()

	k8sAgent.Run(ctx, "Hello") // initiate chat

	messagesReceived := make(chan string)

	go func() {
		var messages []string
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-k8sAgent.Output:
				if !ok {
					return
				}

				m := msg.(*api.Message)
				if m.Source == api.MessageSourceModel && m.Type == api.MessageTypeText {
					messages = append(messages, m.Payload.(string))
				} else if m.Type == api.MessageTypeUserInputRequest {
					final_message := strings.Join(messages, "\n")
					messagesReceived <- final_message
					messages = []string{}
				}
			}
		}
	}()

	// remove greeting response from channel
	<- messagesReceived

	return &KubectlClient{
		agent: k8sAgent,
		messages: messagesReceived,
	}, nil
}

func (c *KubectlClient) Query(ctx context.Context, query string) string {
	c.agent.Input <- &api.UserInputResponse{Query: query}
	return <- c.messages
}

func (c *KubectlClient) Close() error {
	c.agent.Input <- &api.UserInputResponse{Query: "exit"} // end conversation
	close(c.messages)
	return nil
}
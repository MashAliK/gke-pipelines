package client

import (
	"os"
	"fmt"
    "context"
	"bufio"
	"time"

    "github.com/MashAliK/gke-pipelines/internal/config"
	"github.com/GoogleCloudPlatform/kubectl-ai/gollm"
	"github.com/GoogleCloudPlatform/kubectl-ai/pkg/agent"
	"github.com/GoogleCloudPlatform/kubectl-ai/pkg/api"
	"github.com/GoogleCloudPlatform/kubectl-ai/pkg/tools"
	"github.com/GoogleCloudPlatform/kubectl-ai/pkg/journal"
	"github.com/GoogleCloudPlatform/kubectl-ai/pkg/sessions"
)

type KubectlClient struct {
	agent	*agent.Agent
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

	return &KubectlClient{
		agent: k8sAgent,
	}, nil
}

func (c *KubectlClient) Run(ctx context.Context, query string) error {
	scanner := bufio.NewScanner(os.Stdin)

	c.agent.Run(ctx, query)

	agentExited := make(chan struct{})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-c.agent.Output:
				if !ok {
					return
				}
				fmt.Printf("agent output: %+v\n", msg)

				// Check if agent has exited in RunOnce mode
				if c.agent.Session().AgentState == api.AgentStateExited {
					fmt.Println("Agent has exited, terminating session")
					close(agentExited)
					return
				}

			}
		}
	}()

	go func() {
		for {
			if c.agent.Session().AgentState == api.AgentStateDone {
				time.Sleep(3 * time.Second)
				fmt.Print("Your message: ")
				scanner.Scan()
				query := scanner.Text()
				c.agent.Input <- &api.UserInputResponse{Query: query}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case <-agentExited:
		return nil
	}
}
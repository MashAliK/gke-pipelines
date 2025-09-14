package config

import (
    "os"
    "path/filepath"
    
	"github.com/GoogleCloudPlatform/kubectl-ai/pkg/ui"
)

type KubectlAIOptions struct {
	ProviderID string `json:"llmProvider,omitempty"`
	ModelID    string `json:"model,omitempty"`
	
	SkipPermissions bool `json:"skipPermissions,omitempty"`
	
	EnableToolUseShim bool `json:"enableToolUseShim,omitempty"`
	
	Quiet     bool `json:"quiet,omitempty"`
	MCPServer bool `json:"mcpServer,omitempty"`
	MCPClient bool `json:"mcpClient,omitempty"`
	
	ExternalTools bool `json:"externalTools,omitempty"`
	MaxIterations int  `json:"maxIterations,omitempty"`
	
	MCPServerMode string `json:"mcpServerMode,omitempty"`
	
	SSEndpointPort int `json:"sseEndpointPort,omitempty"`
	
	KubeConfigPath string `json:"kubeConfigPath,omitempty"`

	PromptTemplateFilePath string   `json:"promptTemplateFilePath,omitempty"`
	ExtraPromptPaths       []string `json:"extraPromptPaths,omitempty"`
	TracePath              string   `json:"tracePath,omitempty"`
	RemoveWorkDir          bool     `json:"removeWorkDir,omitempty"`
	ToolConfigPaths        []string `json:"toolConfigPaths,omitempty"`

	UIType ui.Type `json:"uiType,omitempty"`
	
	UIListenAddress string `json:"uiListenAddress,omitempty"`

	
	SkipVerifySSL bool `json:"skipVerifySSL,omitempty"`

	
	ResumeSession string `json:"resumeSession,omitempty"`
	NewSession    bool   `json:"newSession,omitempty"`
	ListSessions  bool   `json:"listSessions,omitempty"`
	DeleteSession string `json:"deleteSession,omitempty"`

	
	ShowToolOutput bool `json:"showToolOutput,omitempty"`
}

func (o *KubectlAIOptions) InitDefaults() {
	o.ProviderID = "gemini"
	o.ModelID = "gemini-2.5-flash"
	
	o.SkipPermissions = false
	o.MCPServer = false
	o.MCPClient = false
	
	o.ExternalTools = false
	
	o.EnableToolUseShim = false
	o.Quiet = false
	o.MCPServer = false
	o.MaxIterations = 20
	o.KubeConfigPath = ""
	o.PromptTemplateFilePath = ""
	o.ExtraPromptPaths = []string{}
	o.TracePath = filepath.Join(os.TempDir(), "kubectl-ai-trace.txt")
	o.RemoveWorkDir = false
	o.ToolConfigPaths = []string{}
	o.UIType = ui.UITypeTerminal
	o.UIListenAddress = "localhost:8888"
	o.SkipVerifySSL = false
	o.MCPServerMode = "stdio"
	o.SSEndpointPort = 9080

	o.ResumeSession = ""
	o.NewSession = false
	o.ListSessions = false
	o.DeleteSession = ""

	o.ShowToolOutput = false
}
package cli

import (
	"fmt"
	"flag"
	"os"
	"context"
	"path/filepath"
	"log"
	"bytes"

	"github.com/MashAliK/gke-pipelines/internal/agent"
    "github.com/MashAliK/gke-pipelines/internal/client"
	"github.com/GoogleCloudPlatform/kubectl-ai/gollm"
	"github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/klog/v2"
)


func Run(ctx context.Context) error {
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	klogFlags.Set("logtostderr", "false")
	klogFlags.Set("log_file", filepath.Join(os.TempDir(), "gke-pipelines.log"))

	defer klog.Flush()

	var opt Options

	opt.initDefaults()

	rootCmd, err := buildRootCommand(&opt)
	if err != nil {
		return err
	}
	rootCmd.PersistentFlags().AddGoFlag(klogFlags.Lookup("v"))
	rootCmd.PersistentFlags().AddGoFlag(klogFlags.Lookup("alsologtostderr"))

	redirectStdLogToKlog()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		return err
	}

	return nil
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func buildRootCommand(opt *Options) (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use: "gke-pipelines",
		Short: "Run a gke-pipelines local MCP server.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCP(cmd.Context(), *opt)
		},
	}

	rootCmd.AddCommand(&cobra.Command{
		Use: "version",
		Short: "Print the version number of gke-pipelines",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("version: %s\ncommit: %s\ndate: %s\n", version, commit, date)
			os.Exit(0)
		},
	})

	if err := opt.bindCLIFlags(rootCmd.Flags()); err != nil {
		return nil, err
	}
	return rootCmd, nil
}


type Options struct {
	KubeConfigPath string `json:"kubeConfigPath,omitempty"`
}

func (opt *Options) bindCLIFlags(f *pflag.FlagSet) error {
	f.StringVar(&opt.KubeConfigPath, "kubeconfig", opt.KubeConfigPath, "path to kubeconfig file")

	return nil
}

func (opt *Options) initDefaults() {
	opt.KubeConfigPath = ""
}

type GKEQueryArguments struct {
	Query	string 	`json:"query" jsonschema:"required,description=The task for the GKE agent to perform"`
}

func runMCP(ctx context.Context, opt Options) error {var llmClient gollm.Client
	var err error

	if err := resolveKubeConfigPath(&opt); err != nil {
		return fmt.Errorf("failed to resolve kubeconfig path: %w", err)
	}

	klog.Info("Application started", "pid", os.Getpid())

	llmClient, err = gollm.NewClient(ctx, "")
	if err != nil {
        return err
    }
    defer llmClient.Close()

	k8sAgent, err := client.NewKubectlClient(ctx, &llmClient)
	if err != nil {
		return err
	}
	defer k8sAgent.Close()	
	
	agent := &agent.Agent{
		LLM:		llmClient,

		Model:		"gemini-2.5-pro",

		Provider: 	"Gemini",

		KubectlAIClient: k8sAgent,
	}
	err = agent.Init(ctx)
	if err != nil {
		return err
	}

	done := make(chan struct{})

	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())
	err = server.RegisterTool(
		"query-gke",
		"Send query to an AI agent related to the user's Google Kubernetes Engine cluster. This tool provide a detailed breakdown by running commands within the Kubernetes cluster, this could include seeing logs of pods or viewing the overall state of the cluster. Conversation history is remembered so follow-up questions can be asked. If a follow-up question is asked provide the response in the query of a subsequent tool call.", 
		func(arguments GKEQueryArguments) (*mcp_golang.ToolResponse, error) {
			message, err := agent.SendMessage(ctx, arguments.Query)
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(message)), err
		},
	)
	if err != nil {
		return err
	}

	err = server.Serve()
	if err != nil {
		return err
	}
	
	<- done
	return err
}

// The following logging is from kubectl-ai for consistency
func resolveKubeConfigPath(opt *Options) error {
	switch {
	case opt.KubeConfigPath != "":
		// Already set from flag or viper env
	case os.Getenv("KUBECONFIG") != "":
		opt.KubeConfigPath = os.Getenv("KUBECONFIG")
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		opt.KubeConfigPath = filepath.Join(home, ".kube", "config")
	}

	if opt.KubeConfigPath != "" {
		p, err := filepath.Abs(opt.KubeConfigPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for kubeconfig file %q: %w", opt.KubeConfigPath, err)
		}
		opt.KubeConfigPath = p
	}

	return nil
}

func redirectStdLogToKlog() {
	log.SetOutput(klogWriter{})
	log.SetFlags(0)
}

type klogWriter struct{}

func (writer klogWriter) Write(data []byte) (n int, err error) {
	message := string(bytes.TrimSuffix(data, []byte("\n")))
	klog.Warning(message)
	return len(data), nil
}
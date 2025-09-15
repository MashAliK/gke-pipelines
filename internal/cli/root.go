package cli

import (
	"fmt"
	"flag"
	"os"
	"context"
	"path/filepath"
	"log"
	"bytes"
	"bufio"

	// "github.com/MashAliK/gke-pipelines/internal/agent"
    "github.com/MashAliK/gke-pipelines/internal/client"
	"github.com/GoogleCloudPlatform/kubectl-ai/gollm"
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
		Short: "A CLI tool using LLMs to help developers write and deploy their work on GKE.",
		RunE: func(cmd *cobra.Command, args[]string) error {
			return runRootCommand(cmd.Context(), *opt, args)
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

func runRootCommand(ctx context.Context, opt Options, args []string) error {
	var err error

	if err := resolveKubeConfigPath(&opt); err != nil {
		return fmt.Errorf("failed to resolve kubeconfig path: %w", err)
	}

	klog.Info("Application started", "pid", os.Getpid())

	var llmClient gollm.Client
	llmClient, err = gollm.NewClient(ctx, "")
	if err != nil {
        return err
    }
    defer llmClient.Close()

	// agents := &agent.Agent{
	// 	LLM:		llmClient,

	// 	Model:		"gemini-2.5-flash",

	// 	Provider: 	"Gemini",
	// }

	// err = agents.Init(ctx)

	k8sAgent, err := client.NewKubectlClient(ctx, &llmClient)
	if err != nil {
		return err
	}
	defer k8sAgent.Close()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter your message (type 'quit' to exit):")
	for {
		fmt.Print("> ") // prompt
		if !scanner.Scan() {
			break
		}
		message := scanner.Text()

		if message == "quit" {
			break
		}
		fmt.Println(k8sAgent.Query(ctx, message))
	}

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
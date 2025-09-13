package main

import (
	"fmt"
	"flag"
	"os"
	"context"
	"path/filepath"
	"log"
	"os/signal"
	"syscall"
	"errors"
	"bytes"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/klog/v2"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func BuildRootCommand(opt *Options) (*cobra.Command, error) {
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

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		<-ctx.Done()
		// restore default behavior for a second signal
		signal.Stop(make(chan os.Signal))
		cancel()
		klog.Flush()
		fmt.Fprintf(os.Stderr, "\nReceived signal, shutting down gracefully... (press Ctrl+C again to force)\n")
	}()

	if err := run(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, err)
		}
		
		if errors.Is(err, context.Canceled) {
			os.Exit(0)
		}
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	klogFlags.Set("logtostderr", "false")
	klogFlags.Set("log_file", filepath.Join(os.TempDir(), "kubectl-ai.log"))

	defer klog.Flush()

	var opt Options

	opt.initDefaults()

	rootCmd, err := BuildRootCommand(&opt)
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
	if err := resolveKubeConfigPath(&opt); err != nil {
		return fmt.Errorf("failed to resolve kubeconfig path: %w", err)
	}

	klog.Info("Application started", "pid", os.Getpid())

	return nil
}

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
	// We trim the trailing newline because klog adds its own.
	message := string(bytes.TrimSuffix(data, []byte("\n")))
	klog.Warning(message)
	return len(data), nil
}
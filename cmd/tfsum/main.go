// Command tfsum is a detailed, color-coded, risk-aware alternative to the
// default `terraform plan` review experience. It runs Terraform's real
// planner internally and renders a structured change report.
package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/Fanatic-zer0/tfsum/internal/cli"
	"github.com/Fanatic-zer0/tfsum/internal/tempdir"
)

func main() {
	ctx, stop := tempdir.SignalContext(context.Background())
	defer stop()

	exitCode := cli.ExitInternal
	root := &cobra.Command{
		Use:   "tfsum",
		Short: "A detailed, color-coded, risk-aware alternative to `terraform plan`.",
		// tfsum parses its own flags (see internal/cli) instead of relying
		// on Cobra/pflag, so that arbitrary Terraform flags (including
		// single-dash flags like -target) can be forwarded unchanged.
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		RunE: func(cmd *cobra.Command, args []string) error {
			exitCode = cli.Run(ctx, args, os.Stdout, os.Stderr)
			return nil
		},
	}

	if err := root.Execute(); err != nil {
		os.Exit(cli.ExitInternal)
	}
	os.Exit(exitCode)
}

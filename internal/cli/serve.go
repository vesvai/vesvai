package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/vesvai/vesvai/internal/server"
)

func (c *CLI) newServeCommand() *cobra.Command {
	var port int
	var host string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP API server",
		Long:  "Start the Vesvai HTTP API server, exposing agent and session endpoints via REST with SSE streaming.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := c.cfg
			if port > 0 {
				cfg.Server.Port = port
			}
			if host != "" {
				cfg.Server.Host = host
			}

			srv := server.New(cfg, c.bus, c.log, c.fs, c.sessions, c.llmMgr, c.mcpMgr, c.lspMgr)

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			go func() {
				<-ctx.Done()
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				srv.Shutdown(shutdownCtx)
			}()

			if err := srv.ListenAndServe(); err != nil {
				return fmt.Errorf("serve: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "port to listen on (overrides config)")
	cmd.Flags().StringVar(&host, "host", "", "host to bind to (overrides config)")

	return cmd
}

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	dohttpstd "github.com/samber/do/http/std/v2"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"github.com/whicu/hsa/internal/config"
	"github.com/whicu/hsa/internal/di"
)

const (
	ExitOK          = 0
	ExitGeneral     = 1
	ExitNoInput     = 66 // Missing config file
	ExitUnavailable = 69 // DB/Network unavailable
	ExitSoftware    = 70 // Internal application error
	ExitConfig      = 78 // Invalid configuration
)

func main() {
	if err := cmdRun(); err != nil {
		log.Fatal(err)
	}
}

func cmdRun() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd := &cli.Command{
		Name:                   "hsa",
		Usage:                  "HSA Backend Service",
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "./config/config.yaml",
				Usage:   "Path to YAML configuration file",
				Sources: cli.EnvVars("CONFIG_PATH", "APP_CONFIG"),
			},
			&cli.BoolFlag{
				Name:    "cfg-view",
				Aliases: []string{"v"},
				Usage:   "Print the resolved configuration on startup",
				Sources: cli.EnvVars("CFG_VIEW"),
			},
			&cli.BoolFlag{
				Name:    "count-invites",
				Aliases: []string{"i"},
				Usage:   "Number of root invite codes to generate on startup",
			},
			&cli.BoolFlag{
				Name:    "samber-web-debug",
				Usage:   "Enable samber/do web inspector UI",
				Sources: cli.EnvVars("SAMBER_WEB_DEBUG"),
			},
			&cli.StringFlag{
				Name:    "debug-addr",
				Value:   ":8081",
				Usage:   "HTTP address for samber/do debug UI",
				Sources: cli.EnvVars("DEBUG_ADDR"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			configPath := cmd.String("config")
			fsys, file, err := config.ResolveDiskFS(configPath)
			if err != nil {
				return cli.Exit(err, ExitNoInput)
			}
			injector := di.New(ctx, fsys, file)

			if cmd.Bool("samber-web-debug") {
				go startDebugServer(ctx, injector, cmd.String("debug-addr"))
			}

			debug := do.ExplainInjector(injector)
			fmt.Println(debug.String())
			fmt.Printf("NumCPU=%d GOMAXPROCS=%d\n",
				runtime.NumCPU(),
				runtime.GOMAXPROCS(0),
			)

			runCfg := di.Config{
				CfgView: cmd.Bool("cfg-view"),
				RootInvites: struct {
					CountInvites int
					Out          io.Writer
				}{
					CountInvites: cmd.Count("count-invites"),
					Out:          os.Stdout,
				},
			}

			if errRun := di.Run(ctx, injector, runCfg); errRun != nil {
				return cli.Exit(errRun, ExitSoftware)
			}
			return nil
		},
	}
	if err := cmd.Run(ctx, os.Args); err != nil {
		return err
	}
	return nil
}

func startDebugServer(_ context.Context, injector *do.RootScope, addr string) {
	mux := http.NewServeMux()
	mux.Handle("/debug/do/", dohttpstd.Use("/debug/do", injector))

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	fmt.Printf("Debug UI available at: http://localhost%s/debug/do\n", addr)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("debug server error: %v\n", err)
	}
}

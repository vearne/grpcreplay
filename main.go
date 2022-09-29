package main

import (
	"flag"
	"github.com/vearne/grpcreplay/biz"
	"github.com/vearne/grpcreplay/config"
	slog "github.com/vearne/simplelog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var settings config.AppSettings

func init() {
	flag.DurationVar(&settings.ExitAfter, "exit-after", 0, "exit after specified duration")

	// #################### input ######################
	flag.Var(&config.MultiOption{&settings.InputRAW}, "input-raw",
		`Capture traffic from given port (use RAW sockets and require *sudo* access):
                # Capture traffic from 8080 port
                grpcr --input-raw :8080 --output-grpc grpc://xxx.com
               `)

	flag.Var(&config.MultiOption{&settings.InputFile}, "input-file", "Read requests from file")
	flag.BoolVar(&settings.InputFileLoop, "input-file-loop", false, "")
	flag.IntVar(&settings.InputFileReadDepth, "input-file-read-depth", 100, "")
	// #################### output ######################
	flag.BoolVar(&settings.OutputStdout, "output-stdout", false,
		"Used for testing inputs. Just prints to console data coming from inputs.")

	flag.Var(&config.MultiOption{&settings.OutputGRPC}, "output-grpc",
		`Forwards incoming requests to given grpc address.
			    # Redirect all incoming requests to xxx.com address
                grpcr --input-raw :80 --output-grpc grpc://xxx.com")`)

	flag.Var(&config.MultiOption{&settings.OutputFile},
		"output-file",
		`Write incoming requests to file: 
		        grpcr --input-raw :80 --output-file ./requests.gor`)
}

func main() {
	// set log level
	slog.SetLevel(slog.DebugLevel)

	flag.Parse()
	printSettings(&settings)

	emitter := biz.NewEmitter()
	slog.Debug("--1--")

	plugins := biz.NewPlugins(&settings)
	slog.Debug("plugins:%v", plugins)

	go emitter.Start(plugins)

	closeCh := make(chan int)
	if settings.ExitAfter > 0 {
		slog.Info("Running gor for a duration of %s\n", settings.ExitAfter)

		time.AfterFunc(settings.ExitAfter, func() {
			slog.Info("run timeout %s\n", settings.ExitAfter)
			close(closeCh)
		})
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	exit := 0
	select {
	case <-c:
		exit = 1
	case <-closeCh:
		exit = 0
	}
	//emitter.Close()
	os.Exit(exit)
}

func printSettings(settings *config.AppSettings) {
	slog.Debug("input-raw, %v", settings.InputRAW)
	slog.Debug("output-stdout, %v", settings.OutputStdout)
	slog.Debug("output-file, %v", settings.OutputFile)
	slog.Debug("output-grpc, %v", settings.OutputGRPC)
}

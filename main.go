package main

import (
	"flag"
	"fmt"
	"github.com/vearne/grpcreplay/biz"
	"github.com/vearne/grpcreplay/config"
	"github.com/vearne/grpcreplay/consts"
	"github.com/vearne/grpcreplay/http2"
	"github.com/vearne/grpcreplay/util"
	slog "github.com/vearne/simplelog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const banner string = `
   ______ ____   ____   ______ ____ 
  / ____// __ \ / __ \ / ____// __ \
 / / __ / /_/ // /_/ // /    / /_/ /
/ /_/ // _, _// ____// /___ / _, _/ 
\____//_/ |_|/_/     \____//_/ |_|   

`

var settings config.AppSettings
var version bool

// init registers all command-line flags for configuring the grpcreplay tool, including input and output sources, filtering, rate limiting, proto file paths, and operational timeouts.
func init() {
	flag.BoolVar(&version, "version", false,
		"print version")

	flag.DurationVar(&settings.ExitAfter, "exit-after", 0, "exit after specified duration")

	// #################### input ######################
	flag.Var(&config.MultiStringOption{Params: &settings.InputRAW}, "input-raw",
		`Capture traffic from given port (use RAW sockets and require *sudo* access):
                # Capture traffic from 80 port
                grpcr --input-raw="0.0.0.0:80" --output-grpc="grpc://xx.xx.xx.xx:35001"
               `)

	// input-file-directory
	flag.Var(&config.MultiStringOption{Params: &settings.InputFileDir}, "input-file-directory",
		`grpcr --input-file-directory="/tmp/mycapture" --output-grpc="grpc://xx.xx.xx.xx:35001“`)

	flag.IntVar(&settings.InputFileReadDepth, "input-file-read-depth", 100, "")
	/*
		Replay at 2x speed
		--input-file-replay-speed=2
	*/
	flag.Float64Var(&settings.InputFileReplaySpeed, "input-file-replay-speed", 1, "")

	// input-rocketmq
	flag.Var(&config.MultiStringOption{Params: &settings.InputRocketMQNameServer},
		"input-rocketmq-name-server",
		`grpcr --input-rocketmq-name-server="192.168.2.100:9876" --output-grpc="grpc://xx.xx.xx.xx:35001"`)

	flag.StringVar(&settings.InputRocketMQTopic, "input-rocketmq-topic",
		"test", "")

	flag.StringVar(&settings.InputRocketMQAccessKey, "input-rocketmq-access-key",
		"", "")

	flag.StringVar(&settings.InputRocketMQSecretKey, "input-rocketmq-secret-key",
		"", "")

	flag.StringVar(&settings.InputRocketMQGroupName, "input-rocketmq-group-name",
		"fakeGroupName", "")

	// #################### output ######################
	flag.BoolVar(&settings.OutputStdout, "output-stdout", false,
		"Just prints data to console")

	flag.Var(&config.MultiStringOption{Params: &settings.OutputGRPC}, "output-grpc",
		`Forwards incoming requests to given grpc address.
			    # Redirect all incoming requests to xxx.com address
                grpcr --input-raw="0.0.0.0:80" --output-grpc="grpc://xx.xx.xx.xx:35001")`)

	flag.IntVar(&settings.OutputGRPCWorkerNumber, "output-grpc-worker-number", 5,
		"multiple workers call services concurrently")

	flag.Var(&config.MultiStringOption{Params: &settings.OutputFileDir},
		"output-file-directory",
		`Write incoming requests to file:
		        grpcr --input-raw="0.0.0.0:80" --output-file-directory="/tmp/mycapture"`)

	flag.IntVar(&settings.OutputFileMaxSize, "output-file-max-size", 500,
		"MaxSize is the maximum size in megabytes of the log file before it gets rotated.")

	flag.IntVar(&settings.OutputFileMaxBackups, "output-file-max-backups", 10,
		"MaxBackups is the maximum number of old log files to retain.")

	flag.IntVar(&settings.OutputFileMaxAge, "output-file-max-age", 30,
		`MaxAge is the maximum number of days to retain old log files 
				based on the timestamp encoded in their filename`)

	flag.StringVar(&settings.Codec, "codec", "simple", "")

	flag.StringVar(&settings.IncludeFilterMethodMatch, "include-filter-method-match", "",
		`filter requests when the method matches the specified regular expression`)

	// rate limit
	flag.IntVar(&settings.RateLimitQPS, "rate-limit-qps", -1,
		`the capture rate per second limit for Query`)

	// rocketmq
	flag.Var(&config.MultiStringOption{Params: &settings.OutputRocketMQNameServer},
		"output-rocketmq-name-server",
		`grpcr --input-raw="0.0.0.0:80" --output-rocketmq-name-server="192.168.2.100:9876"`)

	flag.StringVar(&settings.OutputRocketMQTopic, "output-rocketmq-topic",
		"test", "")

	flag.StringVar(&settings.OutputRocketMQAccessKey, "output-rocketmq-access-key",
		"", "")

	flag.StringVar(&settings.OutputRocketMQSecretKey, "output-rocketmq-secret-key",
		"", "")

	flag.BoolVar(&settings.RecordResponse, "record-response", false,
		"record response")

	flag.StringVar(&settings.ProtoFileStr, "proto", "",
		"(optional) proto source file or the directory containing the proto file.")

	flag.DurationVar(&settings.WaitDefaultDuration, "wait-timeout", time.Second,
		`If the output has been processed, the maximum time to wait for the input to be processed
				--wait-timeout=3s			
	`)
}

// main is the entry point for the grpcreplay command-line tool, initializing configuration, setting up components, and running the main event loop until termination or timeout.
func main() {
	fmt.Print(banner)

	// use environment variables to set log level
	logLevel := os.Getenv("SIMPLE_LOG_LEVEL")
	if len(logLevel) <= 0 {
		slog.SetLevel(slog.InfoLevel)
	}

	flag.Parse()
	if version {
		fmt.Println("service: grpcreplay")
		fmt.Println("Version", consts.Version)
		fmt.Println("BuildTime", consts.BuildTime)
		fmt.Println("GitTag", consts.GitTag)
		return
	}

	parseSettings(&settings)
	printSettings(&settings)

	filterChain, err := biz.NewFilterChain(&settings)
	if err != nil {
		slog.Fatal("create FilterChain error:%v", err)
	}
	limiter := biz.NewRateLimit(&settings)
	emitter := biz.NewEmitter(filterChain, limiter)
	plugins := biz.NewPlugins(&settings)

	slog.Info("plugins:%v", plugins)

	go emitter.Start(plugins)

	closeCh := make(chan int)
	if settings.ExitAfter > 0 {
		slog.Info("Running grpcr for a duration of %s\n", settings.ExitAfter)

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

// parseSettings processes the proto file path in the application settings, populates the list of proto files, and sets the default HTTP/2 wait timeout.
// If the proto file path is a directory, all files within it are added; if it is a file, only that file is used.
// Terminates the application with a fatal log if the specified path does not exist or cannot be accessed.
func parseSettings(settings *config.AppSettings) {
	settings.ProtoFileStr = strings.TrimSpace(settings.ProtoFileStr)
	if len(settings.ProtoFileStr) <= 0 {
		return
	}
	// get path information
	fileInfo, err := os.Stat(settings.ProtoFileStr)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Fatal("path [%s] does not exist", settings.ProtoFileStr)
		} else {
			slog.Fatal("error occurred while checking path %s, %v", settings.ProtoFileStr, err)
		}
	}

	fileSet := util.NewStringSet()
	if fileInfo.IsDir() {
		err := util.ListFilesRecursively(settings.ProtoFileStr, fileSet)
		if err != nil {
			slog.Fatal("failed to obtain proto:%v", err)
		}
		settings.ProtoFiles = fileSet.ToArray()
	} else {
		settings.ProtoFiles = []string{settings.ProtoFileStr}
	}

	http2.WaitDefaultDuration = settings.WaitDefaultDuration
}

// printSettings logs the current application configuration settings for input, output, proto files, and wait timeout.
func printSettings(settings *config.AppSettings) {
	slog.Info("input-raw, %v", settings.InputRAW)
	slog.Info("input-file-directory, %v", settings.InputFileDir)
	slog.Info("input-file-replay-speed, %v", settings.InputFileReplaySpeed)

	slog.Info("input-rocketmq-name-server, %v", settings.InputRocketMQNameServer)
	slog.Info("input-rocketmq-topic, %v", settings.InputRocketMQTopic)

	slog.Info("output-stdout, %v", settings.OutputStdout)
	slog.Info("output-file-directory, %v", settings.OutputFileDir)
	slog.Info("output-grpc, %v", settings.OutputGRPC)
	slog.Info("output-rocketmq-name-server, %v", settings.OutputRocketMQNameServer)
	slog.Info("output-rocketmq-topic, %v", settings.OutputRocketMQTopic)

	slog.Info("record-response, %v", settings.RecordResponse)

	if len(settings.ProtoFileStr) > 0 {
		slog.Info("ProtoFileStr, %v", settings.ProtoFileStr)
		slog.Info("ProtoFiles, %v", settings.ProtoFiles)
	}

	slog.Info("wait-timeout, %v", settings.WaitDefaultDuration)
}

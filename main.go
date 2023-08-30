package main

import (
	"flag"
	"fmt"
	"github.com/vearne/grpcreplay/biz"
	"github.com/vearne/grpcreplay/config"
	"github.com/vearne/grpcreplay/consts"
	slog "github.com/vearne/simplelog"
	"os"
	"os/signal"
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
}

func main() {
	fmt.Print(banner)

	adjustLogLevel()

	flag.Parse()
	if version {
		fmt.Println("service: grpcreplay")
		fmt.Println("Version", consts.Version)
		fmt.Println("BuildTime", consts.BuildTime)
		fmt.Println("GitTag", consts.GitTag)
		return
	}

	printSettings(&settings)

	filterChain, err := biz.NewFilterChain(&settings)
	if err != nil {
		slog.Fatal("create FilterChain error:%v", err)
	}
	emitter := biz.NewEmitter(filterChain)
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
}

func adjustLogLevel() {
	logLevel := os.Getenv("SIMPLE_LOG_LEVEL")
	if len(logLevel) > 0 {
		return
	}
	slog.SetLevel(slog.InfoLevel)
}

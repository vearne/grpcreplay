package biz

import (
	"fmt"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/vearne/grpcreplay/config"
	"github.com/vearne/grpcreplay/http2"
	"github.com/vearne/grpcreplay/plugin"
	"github.com/vearne/grpcreplay/util"
	slog "github.com/vearne/simplelog"
	"net"
	"net/url"
	"strings"
)

// InOutPlugins struct for holding references to plugins
type InOutPlugins struct {
	Inputs  []PluginReader
	Outputs []PluginWriter
	All     []interface{}
}

// NewPlugins specify and initialize all available plugins
func NewPlugins(settings *config.AppSettings) *InOutPlugins {
	//  get proto from files
	var finder http2.PBFinder
	if len(settings.ProtoFiles) > 0 {
		finder = http2.NewFilePBFinder(settings.ProtoFiles)
	}

	plugins := new(InOutPlugins)

	for _, item := range settings.InputRAW {
		slog.Debug("options: %q", item)
		host, port, err := net.SplitHostPort(item)
		if err != nil {
			slog.Warn("net.SplitHostPort:%v", err)
			continue
		}
		if finder == nil {
			finder = http2.NewReflectionPBFinder(findOneServerAddr(host, port))
		}
		p, err := plugin.NewRAWInput(item, settings.RecordResponse, finder)
		if err != nil {
			slog.Fatal("NewRAWInput:%v", err)
		}
		plugins.Inputs = append(plugins.Inputs, p)
		plugins.All = append(plugins.All, p)
	}

	for _, path := range settings.InputFileDir {
		err := plugin.IsValidDir(path)
		if err != nil {
			slog.Fatal("%v", err)
		}
		slog.Debug("NewFileDirInput, path:%v", path)
		p := plugin.NewFileDirInput(settings.Codec, path,
			settings.InputFileReadDepth, settings.InputFileReplaySpeed)
		plugins.Inputs = append(plugins.Inputs, p)
		plugins.All = append(plugins.All, p)
	}

	if len(settings.InputRocketMQNameServer) > 0 {
		p, err := plugin.NewRocketMQInput(settings.InputRocketMQNameServer,
			settings.InputRocketMQTopic, settings.InputRocketMQGroupName,
			settings.InputRocketMQAccessKey, settings.InputRocketMQSecretKey)
		if err != nil {
			slog.Fatal("NewRocketMQInput:%v", err)
		}
		plugins.Inputs = append(plugins.Inputs, p)
		plugins.All = append(plugins.All, p)
	}
	// ----------output----------
	if settings.OutputStdout {
		slog.Debug("NewStdOutput")
		p := plugin.NewStdOutput(settings.Codec)
		plugins.Outputs = append(plugins.Outputs, p)
		plugins.All = append(plugins.All, p)
	}

	if len(settings.OutputRocketMQNameServer) > 0 {
		p, err := plugin.NewRocketMQOutput(settings.OutputRocketMQNameServer,
			settings.OutputRocketMQTopic, settings.OutputRocketMQAccessKey, settings.OutputRocketMQSecretKey)
		if err != nil {
			slog.Fatal("NewRocketMQOutput:%v", err)
		}
		plugins.Outputs = append(plugins.Outputs, p)
		plugins.All = append(plugins.All, p)
	}

	for _, item := range settings.OutputGRPC {
		addr, err := extractAddr(item)
		if err != nil {
			slog.Fatal("OutputGRPC addr error:%v", err)
		}
		if finder == nil {
			finder = http2.NewReflectionPBFinder(addr)
		}
		p := plugin.NewGRPCOutput(addr, settings.OutputGRPCWorkerNumber, finder)
		plugins.Outputs = append(plugins.Outputs, p)
		plugins.All = append(plugins.All, p)
	}

	for _, path := range settings.OutputFileDir {
		err := plugin.IsValidDir(path)
		if err != nil {
			slog.Fatal("%v", err)
		}
		cf := &plugin.FileDirOutputConfig{
			MaxSize:    settings.OutputFileMaxSize,
			MaxBackups: settings.OutputFileMaxBackups,
			MaxAge:     settings.OutputFileMaxAge,
		}
		p := plugin.NewFileDirOutput(settings.Codec, path, cf)
		plugins.Outputs = append(plugins.Outputs, p)
		plugins.All = append(plugins.All, p)
	}

	return plugins
}

func extractAddr(outputGrpc string) (string, error) {
	if !strings.Contains(outputGrpc, "grpc://") {
		outputGrpc = "grpc://" + outputGrpc
	}
	u, err := url.Parse(outputGrpc)
	if err != nil {
		return "nil", err
	}
	return u.Host, nil
}

func (plugins *InOutPlugins) String() string {
	return fmt.Sprintf("#####  len(Inputs):%d, len(Outputs):%d, len(All):%d   #####",
		len(plugins.Inputs), len(plugins.Outputs), len(plugins.All))
}

func findOneServerAddr(host string, port string) string {
	if len(host) <= 0 {
		itfStatList, err := psnet.Interfaces()
		if err != nil {
			panic(err)
		}
		for _, itf := range itfStatList {
			for _, addr := range itf.Addrs {
				idx := strings.LastIndex(addr.Addr, "/")
				ip := addr.Addr[0:idx]
				if util.IsIPv4(ip) {
					return fmt.Sprintf("%v:%v", ip, port)
				}
			}
		}
	}
	return fmt.Sprintf("%v:%v", host, port)
}

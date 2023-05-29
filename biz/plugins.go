package biz

import (
	"fmt"
	"github.com/vearne/grpcreplay/config"
	"github.com/vearne/grpcreplay/plugin"
	slog "github.com/vearne/simplelog"
	"net/url"
	"reflect"
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
	plugins := new(InOutPlugins)

	for _, item := range settings.InputRAW {
		slog.Debug("options: %q", item)
		plugins.registerPlugin(plugin.NewRAWInput, item)
	}

	for _, path := range settings.InputFileDir {
		err := plugin.IsValidDir(path)
		if err != nil {
			slog.Fatal("%v", err)
		}
		slog.Debug("NewFileDirInput, path:%v", path)
		plugins.registerPlugin(plugin.NewFileDirInput, settings.Codec, path, settings.InputFileReadDepth)
	}

	if len(settings.InputRocketMQNameServer) > 0 {
		plugins.registerPlugin(plugin.NewRocketMQInput, settings.InputRocketMQNameServer,
			settings.InputRocketMQTopic, settings.InputRocketMQGroupName,
			settings.InputRocketMQAccessKey, settings.InputRocketMQSecretKey)
	}
	// ----------output----------
	if settings.OutputStdout {
		slog.Debug("NewStdOutput")
		plugins.registerPlugin(plugin.NewStdOutput, settings.Codec)
	}

	if len(settings.OutputRocketMQNameServer) > 0 {
		plugins.registerPlugin(plugin.NewRocketMQOutput, settings.OutputRocketMQNameServer,
			settings.OutputRocketMQTopic, settings.OutputRocketMQAccessKey, settings.OutputRocketMQSecretKey)
	}

	for _, item := range settings.OutputGRPC {
		addr, err := extractAddr(item)
		if err != nil {
			slog.Fatal("OutputGRPC addr error:%v", err)
		}
		plugins.registerPlugin(plugin.NewGRPCOutput, addr)
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
		plugins.registerPlugin(plugin.NewFileDirOutput, settings.Codec, path, cf)
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

// Automatically detects type of plugin and initialize it
func (plugins *InOutPlugins) registerPlugin(constructor interface{}, options ...interface{}) {

	vc := reflect.ValueOf(constructor)

	// Pre-processing options to make it work with reflect
	vo := []reflect.Value{}
	for _, oi := range options {
		vo = append(vo, reflect.ValueOf(oi))
	}

	// Calling our constructor with list of given options
	plugin := vc.Call(vo)[0].Interface()

	// Some of the output can be Readers as well because return responses
	if r, ok := plugin.(PluginReader); ok {
		plugins.Inputs = append(plugins.Inputs, r)
	}

	if w, ok := plugin.(PluginWriter); ok {
		plugins.Outputs = append(plugins.Outputs, w)
	}
	plugins.All = append(plugins.All, plugin)
}

func (plugins *InOutPlugins) String() string {
	return fmt.Sprintf("#####  len(Inputs):%d, len(Outputs):%d, len(All):%d   #####",
		len(plugins.Inputs), len(plugins.Outputs), len(plugins.All))
}

package biz

import (
	"fmt"
	"github.com/vearne/grpcreplay/config"
	"github.com/vearne/grpcreplay/plugin"
	slog "github.com/vearne/simplelog"
	"reflect"
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

	//for _, options := range settings.InputFile {
	//	plugins.registerPlugin(plugin.NewFileInput, options, settings.InputFileLoop, settings.InputFileReadDepth)
	//}
	//
	//if settings.OutputStdout {
	//	plugins.registerPlugin(plugin.NewDummyOutput)
	//}

	//for _, path := range settings.OutputFile {
	//	plugins.registerPlugin(plugin.NewFileOutput, path, &settings.OutputFileConfig)
	//}

	//for _, options := range settings.OutputGRPC {
	//	plugins.registerPlugin(NewHTTPOutput, options, &settings.OutputHTTPConfig)
	//}

	return plugins
}

// Automatically detects type of plugin and initialize it
//
// See this article if curious about reflect stuff below: http://blog.burntsushi.net/type-parametric-functions-golang
func (plugins *InOutPlugins) registerPlugin(constructor interface{}, options ...interface{}) {

	vc := reflect.ValueOf(constructor)

	// Pre-processing options to make it work with reflect
	vo := []reflect.Value{}
	for _, oi := range options {
		vo = append(vo, reflect.ValueOf(oi))
		slog.Debug("registerPlugin-%q", oi)
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
	return fmt.Sprintf("len(Inputs):%d, len(Outputs):%d",
		len(plugins.Inputs), len(plugins.Outputs))
}

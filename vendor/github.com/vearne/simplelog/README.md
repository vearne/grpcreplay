### simplelog

### Overview
Simplelog is a simple encapsulation of the standard library "log"
Because I need to be able to control the level of logging.

### Install
```
go get github.com/vearne/simplelog
```

### Use environment variables to set log level
optional value: debug | info | warn | error
```
export SIMPLE_LOG_LEVEL=info
```


### Usage

```
package main

import (
	slog "github.com/vearne/simplelog"
	"os"
)

func main() {
	logFile := "/var/log/simple.log"
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		slog.Fatal("error opening file: %v", err)
	}
	slog.SetOutput(f)
	defer f.Close()
	// set log level
	slog.SetLevel(slog.DebugLevel)

	slog.Debug("log, %v", "debug")
	slog.Info("log, %v", "info")
	slog.Error("log, %v", "error")
	// Like log, Fatal() will terminal process
	slog.Fatal("log, %v", "fatal")
}
```

### Output
```
2020/05/27 17:40:01 [debug] log, debug
2020/05/27 17:40:01 [info] log, info
2020/05/27 17:40:01 [error] log, error
2020/05/27 17:40:01 [fatal] log, fatal
```


# timer
[![golang-ci](https://github.com/vearne/gtimer/actions/workflows/golang-ci.yml/badge.svg)](https://github.com/vearne/gtimer/actions/workflows/golang-ci.yml)

Timer based on delayqueue

* [中文 README](https://github.com/vearne/gtimer/blob/master/README_zh.md)

# Design and implementation
The implementation is inspired by Java DelayQueue.java
Portal:
[DelayQueue.java](http://www.docjar.com/html/api/java/util/concurrent/DelayQueue.java.html)

The dependent structures are as follows:

timer -> delayqueue -> priorityqueue -> heap

# Installation
## Install:

```
go get -u github.com/vearne/gtimer
```
## Import:
```
import "github.com/vearne/gtimer"
```


## Quick Start
[more examples](https://github.com/vearne/gtimer/blob/master/example)
```
package main

import (
	"fmt"
	"github.com/vearne/gtimer"
	"time"
)

func main() {
	startTime := time.Now()
	gtimer.Add(3*time.Second, func() {
		fmt.Println(time.Since(startTime))
	})
	go func() {
		time.Sleep(5 * time.Second)
		gtimer.Stop()
	}()
	gtimer.Wait()
	fmt.Println("---end---")
}
```

## Performance

`CPU Model Name`: 2.3 GHz Intel Core i5     
`CPU Processors`: 4     
`Memory`: 8GB

### Benchmark Test Results


| produce goroutines count | consume goroutines count | qps(per second) | 
| ---------:| ----------:| --------:| 
| 1| 1                      | 285714             |  
| 10| 10                    | 90090                |  
| 10| 100                   | 89285              |  
| 100| 100                  | 23255              |  








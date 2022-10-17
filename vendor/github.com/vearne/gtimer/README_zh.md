# timer
[![golang-ci](https://github.com/vearne/gtimer/actions/workflows/golang-ci.yml/badge.svg)](https://github.com/vearne/gtimer/actions/workflows/golang-ci.yml)

基于delayqueue实现的定时器

* [English README](https://github.com/vearne/gtimer/blob/master/README.md)

# 实现
实现受到了Java DelayQueue.java的启发
源码地址
[DelayQueue.java](http://www.docjar.com/html/api/java/util/concurrent/DelayQueue.java.html)

依赖的几个结构依次为为
timer -> delayqueue -> priorityqueue -> heap

# 安装
## Install:

```
go get -u github.com/vearne/gtimer
```
## Import:
```
import "github.com/vearne/gtimer"
```


## 快速开始
[更多示例](https://github.com/vearne/gtimer/blob/master/example)
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

## 性能

`CPU Model Name`: 2.3 GHz Intel Core i5     
`CPU Processors`: 4     
`Memory`: 8GB

### 压测结果


| 生产者数量 | 消费者数量 | QPS | 
| ---------:| ----------:| --------:| 
| 1| 1                      | 285714             |  
| 10| 10                    | 90090                |  
| 10| 100                   | 89285              |  
| 100| 100                  | 23255              |  






package log

import (
	"io/ioutil"
	"log"
	"os"
	"sync"
)

var (
	// 控制台彩色前缀的 error/info 日志器，默认输出到 Stdout 并带时间+短文件名
	//[info ] 颜色为蓝色，[error] 为红色。
	errorLog = log.New(os.Stdout, "\033[31m[error ]\033[0m ", log.LstdFlags|log.Lshortfile)
	infoLog  = log.New(os.Stdout, "\033[34m[info ]\033[0m ", log.LstdFlags|log.Lshortfile)
	loggers  = []*log.Logger{infoLog, errorLog}
	mu       sync.Mutex
)

// log methods
var (
	Error  = errorLog.Println
	Errorf = errorLog.Printf
	Info   = infoLog.Println
	Infof  = infoLog.Printf
)

// log levels
const (
	InfoLevel = iota
	ErrorLevel
	Disabled
)

// SetLogLevel 设置日志输出级别
func SetLogLevel(level int) {
	mu.Lock()
	defer mu.Unlock()

	// 先将所有 logger 重置为输出到 Stdout，作为基线
	for _, logger := range loggers {
		logger.SetOutput(os.Stdout)
	}

	// 按照目标 level 屏蔽更“低优先级”的日志：
	// - 当设置为 ErrorLevel：屏蔽 Info，仅保留 Error
	// - 当设置为 Disabled：Info 与 Error 均屏蔽
	if ErrorLevel < level {
		errorLog.SetOutput(ioutil.Discard)
	}

	if InfoLevel < level {
		infoLog.SetOutput(ioutil.Discard)
	}
}

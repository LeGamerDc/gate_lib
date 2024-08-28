package gate

import (
	"runtime"

	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type Config struct {
	LoopCount int                // 设置几个epoll loop用于处理网络消息，[1, runtime.NumCPU()]
	Port      int                // 设置监听端口
	CHB       ConnHandlerBuilder // 设置消息处理逻辑的Builder，不能为空
	SB        SenderBuilder      // 设置发送逻辑，建议使用DefaultSenderBuilder，不能为空
	Logger    logging.Logger     // 给gate以及gnet提供日志接口，不能为空
}

func (c *Config) purge() {
	if c.LoopCount < 1 {
		c.LoopCount = 1
	}
	if c.LoopCount > runtime.NumCPU() {
		c.LoopCount = runtime.NumCPU()
	}

	if c.CHB == nil || c.SB == nil {
		panic("builder must set")
	}
	if c.Logger == nil {
		panic("logger must set")
	}
}

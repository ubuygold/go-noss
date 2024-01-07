package noss_chain

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const ReportURL = "wss://report-worker-2.noscription.org"

type NossChain struct {
	conn    *websocket.Conn
	eventID atomic.Value

	ready int32
}

func NewNossChain() *NossChain {
	wait := &sync.WaitGroup{}
	wait.Add(1)
	return &NossChain{
		conn: connecTo(ReportURL),
	}
}

func (c *NossChain) ListenEvent(listenFn ...func(event string)) {
	event := &event{}
	for {
		if err := c.conn.ReadJSON(event); err != nil {
			logrus.Errorf("read: %s", err)
			_ = c.conn.Close()
			c.conn = connecTo(ReportURL)
			continue
		}
		c.eventID.Store(event.EventID)

		// callback
		if len(listenFn) > 0 {
			listenFn[0](event.EventID)
		}

		atomic.StoreInt32(&c.ready, 1)
	}
}

func (c *NossChain) GetLatestEventID() string {
	return c.eventID.Load().(string)
}

func (c *NossChain) WaitReady() {
	for c.ready == 0 {
	}
}

func connecTo(url string) *websocket.Conn {
	headers := http.Header{}
	headers.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0")
	headers.Add("Origin", "https://noscription.org")
	headers.Add("Host", "report-worker-2.noscription.org")
	for {
		// 使用gorilla/websocket库建立连接
		conn, _, err := websocket.DefaultDialer.Dial(url, headers)
		logrus.Println("Connecting to wss...")
		if err != nil {
			// 连接失败，打印错误并等待一段时间后重试
			logrus.Errorf("Error connecting to WebSocket: %s", err)
			// time.Sleep(1 * time.Second) // 5秒重试间隔
			continue
		}
		// 连接成功，退出循环
		return conn
	}
}

type event struct {
	EventID string `json:"eventId"`
}

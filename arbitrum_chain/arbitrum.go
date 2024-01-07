package arbitrum_chain

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"sync/atomic"
)

type ArbitrumChain struct {
	url          string
	client       *ethclient.Client
	latestHex    atomic.Value
	latestNumber uint64

	ready int32
}

func NewArbitrumChain(url string) *ArbitrumChain {
	client, err := ethclient.Dial(url)
	if err != nil {
		logrus.Fatalf("无法连接到Arbitrum节点: %v", err)
	}
	return &ArbitrumChain{
		url:    url,
		client: client,
	}
}

func (c *ArbitrumChain) ListenNewHeader() {
	listener := make(chan *types.Header)
	_, err := c.client.SubscribeNewHead(context.Background(), listener)
	if err != nil {
		logrus.Fatalf(fmt.Sprintf("监听%s地址断连: %s", c.url, err))
	}
	go func() {
		for head := range listener {
			fmt.Println(head.Hash().Hex(), head.Number.Uint64())
			c.latestHex.Store(head.Hash().Hex())
			atomic.StoreUint64(&c.latestNumber, head.Number.Uint64())

			atomic.StoreInt32(&c.ready, 1)
		}
	}()
}

func (c *ArbitrumChain) LatestHex() string {
	return c.latestHex.Load().(string)
}

func (c *ArbitrumChain) LatestNumber() uint64 {
	return c.latestNumber
}

func (c *ArbitrumChain) WaitReady() {
	for c.ready == 0 {
	}
}

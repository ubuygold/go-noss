package main

import (
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	arbitrum "nostr/arbitrum_chain"
	"nostr/miner"
	noss "nostr/noss_chain"
	"os"
	"strconv"
)

var sk, pk string
var workers int
var arbRpcURL string

func init() {
	err := godotenv.Load()
	if err != nil {
		logrus.Fatal("Error loading .env file")
	}
	sk = os.Getenv("sk")
	pk = os.Getenv("pk")
	workers, _ = strconv.Atoi(os.Getenv("numberOfWorkers"))
	arbRpcURL = os.Getenv("arbRpcUrl")

	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "15:04:05",
		FullTimestamp:   true,
	})
}

func main() {
	arbitrumChain := arbitrum.NewArbitrumChain(arbRpcURL)
	nossChain := noss.NewNossChain()
	go arbitrumChain.ListenNewHeader()
	go nossChain.ListenEvent()
	logrus.Info("arbitrum chain 与 noss websocket 开始监听...")

	arbitrumChain.WaitReady()
	nossChain.WaitReady()

	for i := 0; i < workers; i++ {
		logrus.Println("启动 Miner", i)
		go miner.NewMiner(arbitrumChain, nossChain, pk, sk).Mining()
	}

	select {}
}

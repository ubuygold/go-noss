package main

import (
	"bytes"
	"context"
	"crypto/rand"

	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip13"
)

var sk string
var pk string
var numberOfWorkers int
var nonceFound int32 = 0
var blockNumber uint64
var hash atomic.Value
var messageId atomic.Value
var currentWorkers int32
var arbRpcUrl string

var (
	ErrDifficultyTooLow = errors.New("nip13: insufficient difficulty")
	ErrGenerateTimeout  = errors.New("nip13: generating proof of work took too long")
)

func init() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	sk = os.Getenv("sk")
	pk = os.Getenv("pk")
	numberOfWorkers, _ = strconv.Atoi(os.Getenv("numberOfWorkers"))
	arbRpcUrl = os.Getenv("arbRpcUrl")
}

func generateRandomString(length int) (string, error) {
	charset := "abcdefghijklmnopqrstuvwxyz0123456789" // 字符集
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	for i := 0; i < length; i++ {
		b[i] = charset[int(b[i])%len(charset)]

	}

	return string(b), nil
}

func Generate(event nostr.Event, targetDifficulty int) (nostr.Event, error) {
	tag := nostr.Tag{"nonce", "", strconv.Itoa(targetDifficulty)}
	event.Tags = append(event.Tags, tag)
	start := time.Now()
	for {
		nonce, err := generateRandomString(10)
		if err != nil {
			fmt.Println(err)
		}
		tag[1] = nonce
		event.CreatedAt = nostr.Now()
		if nip13.Difficulty(event.GetID()) >= targetDifficulty {
			// fmt.Print(time.Since(start))
			return event, nil
		}
		if time.Since(start) >= 1*time.Second {
			return event, ErrGenerateTimeout
		}
	}
}

type Message struct {
	EventId string `json:"eventId"`
}

type EV struct {
	Sig       string          `json:"sig"`
	Id        string          `json:"id"`
	Kind      int             `json:"kind"`
	CreatedAt nostr.Timestamp `json:"created_at"`
	Tags      nostr.Tags      `json:"tags"`
	Content   string          `json:"content"`
	PubKey    string          `json:"pubkey"`
}

func mine(ctx context.Context, messageId string, client *ethclient.Client) {

	replayUrl := "wss://relay.noscription.org/"
	difficulty := 21

	// Create a channel to signal the finding of a valid nonce
	foundEvent := make(chan nostr.Event, 1)
	notFound := make(chan nostr.Event, 1)
	// Create a channel to signal all workers to stop
	content := "{\"p\":\"nrc-20\",\"op\":\"mint\",\"tick\":\"noss\",\"amt\":\"10\"}"
	startTime := time.Now()

	ev := nostr.Event{
		Content:   content,
		CreatedAt: nostr.Now(),
		ID:        "",
		Kind:      nostr.KindTextNote,
		PubKey:    pk,
		Sig:       "",
		Tags:      nil,
	}
	ev.Tags = ev.Tags.AppendUnique(nostr.Tag{"p", "9be107b0d7218c67b4954ee3e6bd9e4dba06ef937a93f684e42f730a0c3d053c"})
	ev.Tags = ev.Tags.AppendUnique(nostr.Tag{"e", "51ed7939a984edee863bfbb2e66fdc80436b000a8ddca442d83e6a2bf1636a95", replayUrl, "root"})
	ev.Tags = ev.Tags.AppendUnique(nostr.Tag{"e", messageId, replayUrl, "reply"})
	ev.Tags = ev.Tags.AppendUnique(nostr.Tag{"seq_witness", strconv.Itoa(int(blockNumber)), hash.Load().(string)})
	// Start multiple worker goroutines
	go func() {
		select {
		case <-ctx.Done():
			return
		default:
			evCopy := ev
			evCopy, err := Generate(evCopy, difficulty)
			if err != nil {
				// fmt.Println(err)
				notFound <- evCopy
			}
			foundEvent <- evCopy
		}
	}()

	select {
	case <-notFound:
	case evNew := <-foundEvent:
		evNew.Sign(sk)

		evNewInstance := EV{
			Sig:       evNew.Sig,
			Id:        evNew.ID,
			Kind:      evNew.Kind,
			CreatedAt: evNew.CreatedAt,
			Tags:      evNew.Tags,
			Content:   evNew.Content,
			PubKey:    evNew.PubKey,
		}
		// 将ev转为Json格式
		eventJSON, err := json.Marshal(evNewInstance)
		if err != nil {
			log.Fatal(err)
		}

		wrapper := map[string]json.RawMessage{
			"event": eventJSON,
		}

		// 将包装后的对象序列化成JSON
		wrapperJSON, err := json.Marshal(wrapper)
		if err != nil {
			log.Fatalf("Error marshaling wrapper: %v", err)
		}

		url := "https://api-worker.noscription.org/inscribe/postEvent"
		// fmt.Print(bytes.NewBuffer(wrapperJSON))
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(wrapperJSON)) // 修改了弱智项目方不识别美化Json的bug
		if err != nil {
			log.Fatalf("Error creating request: %v", err)
		}

		// 设置HTTP Header
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0")
		req.Header.Set("Sec-ch-ua", "\"Not A(Brand\";v=\"99\", \"Microsoft Edge\";v=\"121\", \"Chromium\";v=\"121\"")
		req.Header.Set("Sec-ch-ua-mobile", "?0")
		req.Header.Set("Sec-ch-ua-platform", "\"Windows\"")
		req.Header.Set("Sec-fetch-dest", "empty")
		req.Header.Set("Sec-fetch-mode", "cors")
		req.Header.Set("Sec-fetch-site", "same-site")

		// 发送请求
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Error sending request: %v", err)
		}
		defer resp.Body.Close()

		fmt.Println("Response Status:", resp.Status)
		spendTime := time.Since(startTime)
		// fmt.Println("Response Body:", string(body))
		fmt.Println(nostr.Now().Time(), "spend: ", spendTime, "!!!!!!!!!!!!!!!!!!!!!published to:", evNew.ID)
		atomic.StoreInt32(&nonceFound, 0)
	case <-ctx.Done():
		fmt.Print("done")
	}

}

func connectToWSS(url string) (*websocket.Conn, error) {
	var conn *websocket.Conn
	var err error
	headers := http.Header{}
	headers.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0")
	headers.Add("Origin", "https://noscription.org")
	headers.Add("Host", "report-worker-2.noscription.org")
	for {
		// 使用gorilla/websocket库建立连接
		conn, _, err = websocket.DefaultDialer.Dial(url, headers)
		fmt.Println("Connecting to wss")
		if err != nil {
			// 连接失败，打印错误并等待一段时间后重试
			fmt.Println("Error connecting to WebSocket:", err)
			// time.Sleep(1 * time.Second) // 5秒重试间隔
			continue
		}
		// 连接成功，退出循环
		break
	}
	return conn, nil
}

func main() {

	wssAddr := "wss://report-worker-2.noscription.org"
	// relayUrl := "wss://relay.noscription.org/"
	ctx := context.Background()

	var err error

	client, err := ethclient.Dial(arbRpcUrl)
	if err != nil {
		log.Fatalf("无法连接到Arbitrum节点: %v", err)
	}

	c, err := connectToWSS(wssAddr)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	// initialize an empty cancel function

	// get block
	go func() {
		for {
			header, err := client.HeaderByNumber(context.Background(), nil)
			if err != nil {
				log.Fatalf("无法获取最新区块号: %v", err)
			}
			if header.Number.Uint64() >= blockNumber {
				hash.Store(header.Hash().Hex())
				atomic.StoreUint64(&blockNumber, header.Number.Uint64())
			}
		}
	}()

	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}

			var messageDecode Message
			if err := json.Unmarshal(message, &messageDecode); err != nil {
				fmt.Println(err)
				continue
			}
			messageId.Store(messageDecode.EventId)
		}

	}()

	atomic.StoreInt32(&currentWorkers, 0)
	// 初始化一个取消上下文和它的取消函数
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听blockNumber和messageId变化
	go func() {
		for {
			select {
			case <-ctx.Done(): // 如果上下文被取消，则退出协程
				return
			default:
				if atomic.LoadInt32(&currentWorkers) < int32(numberOfWorkers) && messageId.Load() != nil && blockNumber > 0 {
					atomic.AddInt32(&currentWorkers, 1) // 增加工作者数量
					go func(bn uint64, mid string) {
						defer atomic.AddInt32(&currentWorkers, -1) // 完成后减少工作者数量
						mine(ctx, mid, client)
					}(blockNumber, messageId.Load().(string))
				}
			}
		}
	}()

	select {}

}

package miner

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip13"
	"github.com/sirupsen/logrus"
	"net/http"
	arbitrum "nostr/arbitrum_chain"
	noss "nostr/noss_chain"
	rander "nostr/pkg/rander"
	"nostr/pkg/ring_buffer"
	"strconv"
	"time"
)

const (
	message      = `{"p":"nrc-20","op":"mint","tick":"noss","amt":"10"}`
	replayURL    = "wss://relay.noscription.org/"
	postEventURL = "https://api-worker.noscription.org/inscribe/postEvent"
	difficulty   = 21

	MineTimeout = time.Second * 1
)

var (
	ErrDifficultyTooLow = errors.New("nip13: insufficient difficulty")
	ErrGenerateTimeout  = errors.New("nip13: generating proof of work took too long")
)

type Miner struct {
	arb       *arbitrum.ArbitrumChain
	noss      *noss.NossChain
	publicKey string
	secretKey string

	cache *ring_buffer.RingBuffer[string]
}

func NewMiner(arb *arbitrum.ArbitrumChain, noss *noss.NossChain, publicKey, secretKey string) *Miner {
	return &Miner{
		publicKey: publicKey,
		secretKey: secretKey,
		arb:       arb,
		noss:      noss,
		// 缓存最近提交的200个记录，如果在noss_chain中接收到了，那么应该是自己挖到了
		cache: ring_buffer.NewRingBuffer[string](200),
	}
}

func (m *Miner) Mine() {
	for {
		ev := &nostr.Event{
			Content:   message,
			CreatedAt: nostr.Now(),
			ID:        "",
			Kind:      nostr.KindTextNote,
			PubKey:    m.publicKey,
			Sig:       "",
			Tags:      nil,
		}
		ev.Tags = ev.Tags.AppendUnique(nostr.Tag{"p", "9be107b0d7218c67b4954ee3e6bd9e4dba06ef937a93f684e42f730a0c3d053c"})
		ev.Tags = ev.Tags.AppendUnique(nostr.Tag{"e", "51ed7939a984edee863bfbb2e66fdc80436b000a8ddca442d83e6a2bf1636a95", replayURL, "root"})
		ev.Tags = ev.Tags.AppendUnique(nostr.Tag{"e", m.noss.GetLatestEventID(), replayURL, "reply"})
		ev.Tags = ev.Tags.AppendUnique(nostr.Tag{"seq_witness", strconv.Itoa(int(m.arb.LatestNumber())), m.arb.LatestHex()})

		if event, err := generate(ev, difficulty); err != nil {
			continue
		} else {
			m.postEvent(event)
		}
	}
}

func (m *Miner) postEvent(event *nostr.Event) {
	_ = event.Sign(m.secretKey)

	evNewInstance := EV{
		Sig:       event.Sig,
		Id:        event.ID,
		Kind:      event.Kind,
		CreatedAt: event.CreatedAt,
		Tags:      event.Tags,
		Content:   event.Content,
		PubKey:    event.PubKey,
	}
	// 将ev转为Json格式
	eventJSON, _ := json.Marshal(evNewInstance)

	wrapper := map[string]json.RawMessage{
		"event": eventJSON,
	}

	// 将包装后的对象序列化成JSON
	wrapperJSON, _ := json.Marshal(wrapper)

	req, err := http.NewRequest("POST", postEventURL, bytes.NewBuffer(wrapperJSON)) // 修改了弱智项目方不识别美化Json的bug
	if err != nil {
		logrus.Errorf("Error creating request: %v", err)
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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logrus.Errorf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	logrus.Info("published to: ", event.ID, " Response Status: ", resp.Status)
}

var fastRand = rander.NewRander([]byte("abcdefghijklmnopqrstuvwxyz0123456789"))

func generate(event *nostr.Event, targetDifficulty int) (*nostr.Event, error) {
	tag := nostr.Tag{"nonce", "", strconv.Itoa(targetDifficulty)}
	event.Tags = append(event.Tags, tag)
	start := time.Now()
	nonce := make([]byte, 10)
	for {
		tag[1] = fastRand.Rand(nonce)
		event.CreatedAt = nostr.Now()
		if nip13.Difficulty(event.GetID()) >= targetDifficulty {
			return event, nil
		}
		//if difficulty21(sha256.Sum256(event.Serialize())) {
		//	return event, nil
		//}
		// 超过1秒钟就重新挖
		if time.Since(start) >= MineTimeout {
			return event, ErrGenerateTimeout
		}
	}
}

func difficulty21(hex [32]byte) bool {
	if hex[0] == 0 && hex[1] == 0 && hex[2] <= 5 {
		return true
	} else {
		return false
	}
}

package cluster

import (
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/protocal/codec"
	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/protocal/packet"
)

// 缓存的数据
var (
	hsd            []byte // 握手响应数据, 时间戳需要动态替换
	hbd            []byte // 心跳数据包数据
	hsdPlaceholder = []byte(strconv.FormatInt(math.MaxInt64, 10))
)

// 执行缓存
func cache() {
	hrdata := map[string]any{
		"code": 200,
		"sys": map[string]any{
			"heartbeat":  env.HeartbeatInterval.Seconds(),
			"servertime": math.MaxInt64, //占位符
		},
	}
	if dict, ok := message.GetRouteDict(); ok {
		hrdata = map[string]any{
			"code": 200,
			"sys": map[string]any{
				"heartbeat":  env.HeartbeatInterval.Seconds(),
				"servertime": math.MaxInt64, //占位符
				"dict":       dict,
			},
		}
	}
	// data, err := json.Marshal(map[string]any{
	// 	"code": 200,
	// 	"sys": map[string]float64{
	// 		"heartbeat": env.Heartbeat.Seconds(),
	// 	},
	// })
	data, err := json.Marshal(hrdata)
	if err != nil {
		panic(err)
	}

	hsd, err = codec.Encode(packet.Handshake, data)
	if err != nil {
		panic(err)
	}

	hbd, err = codec.Encode(packet.Heartbeat, nil)
	if err != nil {
		panic(err)
	}
}

// 获取握手响应, 缓存的
func getHsd() []byte {
	servertime := []byte(strconv.FormatInt(time.Now().Unix(), 10))
	return codec.Replace(hsd, hsdPlaceholder, servertime)
}

// 获取心跳请求, 缓存的
func getHbd() []byte {
	return hbd
}

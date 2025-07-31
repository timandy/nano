package cluster

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/protocal/packet"
)

// 缓存的数据
var (
	hsd []byte // 握手响应数据, 时间戳需要动态替换
	hbd []byte // 心跳数据包数据
)

// 占位符, 握手响应数据包中时间戳的占位符
var (
	hsdTimePlaceHolder    = "${Time}"
	hsdTimePlaceHolderKey = []byte("\"" + hsdTimePlaceHolder + "\"")
)

// 执行缓存
func cache() {
	// 握手数据 map
	var hsdmap map[string]any
	if dict, ok := message.GetRouteDict(); ok {
		hsdmap = map[string]any{
			"code": 200,
			"sys": map[string]any{
				"heartbeat":  env.HeartbeatInterval.Seconds(),
				"servertime": hsdTimePlaceHolder, //占位符
				"dict":       dict,
			},
		}
	} else {
		hsdmap = map[string]any{
			"code": 200,
			"sys": map[string]any{
				"heartbeat":  env.HeartbeatInterval.Seconds(),
				"servertime": hsdTimePlaceHolder, //占位符
			},
		}
	}

	//握手数据包内容, json
	hsdata, err := json.Marshal(hsdmap)
	if err != nil {
		panic(err)
	}

	//握手数据包, 已添加包头
	hsd, err = packet.Encode(packet.Handshake, hsdata)
	if err != nil {
		panic(err)
	}

	// 心跳数据包, 已添加包头, 包体为空
	hbd, err = packet.Encode(packet.Heartbeat, nil)
	if err != nil {
		panic(err)
	}
}

// 获取握手数据包, 缓存的
func getHsd() []byte {
	servertime := []byte(strconv.FormatInt(time.Now().Unix(), 10))
	return packet.Replace(hsd, hsdTimePlaceHolderKey, servertime)
}

// 获取心跳数据包, 缓存的
func getHbd() []byte {
	return hbd
}

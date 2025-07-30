// Copyright (c) nano Authors. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// env represents the environment of the current process, includes
// work path and config path etc.

package env

import (
	"time"

	"github.com/lonng/nano/protocal/serialize"
	"github.com/lonng/nano/protocal/serialize/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

//goland:noinspection GoVarAndConstTypeMayBeOmitted,GoCommentStart
var (
	Debug          bool                 = false                    //调试模式
	DieChan        chan bool            = make(chan bool)          //等待停止的 chan
	Serializer     serialize.Serializer = protobuf.NewSerializer() //序列化器, 对象和字节转换
	TimerPrecision time.Duration        = time.Second              //定时器精度
	//集群
	RetryInterval     time.Duration     = 3 * time.Second    //子节点向 Master 注册失败后, 重试间隔时间, 默认 3s
	HeartbeatInterval time.Duration     = 30 * time.Second   //子节点向 Master 定时心跳请求间隔
	GrpcOptions       []grpc.DialOption = []grpc.DialOption{ //子节点的 GRPC 客户端的连接选项
		grpc.WithTransportCredentials(insecure.NewCredentials()),                       //非 TLS
		grpc.WithConnectParams(grpc.ConnectParams{MinConnectTimeout: 2 * time.Second}), //连接参数
	}
)

// 初始化对外暴漏的函数
func init() {
	serialize.Marshal = Marshal
	serialize.Unmarshal = Unmarshal
}

// Marshal 序列化数据
func Marshal(v any) ([]byte, error) {
	switch raw := v.(type) {
	case []byte:
		return raw, nil
	case string:
		return []byte(raw), nil
	case *string:
		if raw == nil {
			return []byte{}, nil
		}
		return []byte(*raw), nil
	default:
		return Serializer.Marshal(v)
	}
}

// Unmarshal 反序列化数据
func Unmarshal(data []byte, v any) error {
	switch raw := v.(type) {
	case *[]byte:
		*raw = data
		return nil
	case *string:
		*raw = string(data)
		return nil
	case **string:
		s := string(data)
		*raw = &s
		return nil
	default:
		return Serializer.Unmarshal(data, v)
	}
}

// Close 关闭 DieChan 通道, 以便其他组件可以监听到
func Close() {
	defer func() {
		recover()
	}()
	close(DieChan)
}

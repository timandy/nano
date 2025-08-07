package snowflake

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/lonng/nano/internal/utils/net"
)

var localRand = rand.New(rand.NewSource(time.Now().UnixNano()))

const (
	//时间起始标记点，作为基准，一般取系统的最近时间（一旦确定不能变动）
	twepoch int64 = 1288834974657
	//机器标识位数
	workerIdBits     int64 = 5
	datacenterIdBits int64 = 5
	maxWorkerId      int64 = -1 ^ (-1 << workerIdBits)
	maxDatacenterId  int64 = -1 ^ (-1 << datacenterIdBits)
	//毫秒内自增位
	sequenceBits      int64 = 12
	workerIdShift     int64 = sequenceBits
	datacenterIdShift int64 = sequenceBits + workerIdBits
	//时间戳左移动位
	timestampLeftShift int64 = sequenceBits + workerIdBits + datacenterIdBits
	sequenceMask       int64 = -1 ^ (-1 << sequenceBits)
)

// SnowFlake 雪花算法
type SnowFlake struct {
	lock          sync.Mutex //锁
	datacenterId  int64      //数据中心id
	workerId      int64      //工作者id
	sequence      int64      //序列号
	lastTimestamp int64      //上一次时间戳
}

// NewSnowFlake 创建一个新的 SnowFlake 实例，自动获取数据中心 ID 和工作者 ID
func NewSnowFlake() *SnowFlake {
	datacenterId := getDatacenterId(maxDatacenterId)
	workerId := getWorkerId(datacenterId, maxWorkerId)
	return &SnowFlake{datacenterId: datacenterId, workerId: workerId}
}

// NewSnowFlakeWithConf 创建一个新的 SnowFlake 实例，使用指定的数据中心 ID 和工作者 ID
func NewSnowFlakeWithConf(datacenterId, workerId int64) *SnowFlake {
	return &SnowFlake{datacenterId: datacenterId, workerId: workerId}
}

// GetDatacenterId 返回数据中心 ID
func (sf *SnowFlake) GetDatacenterId() int64 {
	return sf.datacenterId
}

// GetWorkerId 返回工作者 ID
func (sf *SnowFlake) GetWorkerId() int64 {
	return sf.workerId
}

// NextId 生成下一个唯一 ID
func (sf *SnowFlake) NextId() int64 {
	sf.lock.Lock()
	defer sf.lock.Unlock()

	timestamp := timeGen()
	//回拨或润秒
	if timestamp < sf.lastTimestamp {
		offset := sf.lastTimestamp - timestamp
		if offset > 5 {
			panic(fmt.Sprintf("Clock moved backwards.  Refusing to generate id for %d milliseconds", offset))
		}
		wait(offset << 1)
		timestamp = timeGen()
		if timestamp < sf.lastTimestamp {
			panic(fmt.Sprintf("Clock moved backwards.  Refusing to generate id for %d milliseconds", offset))
		}
	}

	if sf.lastTimestamp == timestamp {
		// 相同毫秒内，序列号自增
		sf.sequence = (sf.sequence + 1) & sequenceMask
		if sf.sequence == 0 {
			// 同一毫秒的序列数已经达到最大
			timestamp = tilNextMillis(sf.lastTimestamp)
		}
	} else {
		// 不同毫秒内，序列号置为 1 - 2 随机数
		sf.sequence = int64(nextInt(1, 3))
	}

	sf.lastTimestamp = timestamp

	// 时间戳部分 | 数据中心部分 | 机器标识部分 | 序列号部分
	return ((timestamp - twepoch) << timestampLeftShift) |
		(sf.datacenterId << datacenterIdShift) |
		(sf.workerId << workerIdShift) |
		sf.sequence
}

// NextIdStr 生成下一个唯一 ID 的字符串表示
func (sf *SnowFlake) NextIdStr() string {
	id := sf.NextId()
	return strconv.FormatInt(id, 10)
}

// NextIdHex 生成下一个唯一 ID 的十六进制表示
func (sf *SnowFlake) NextIdHex() string {
	id := sf.NextId()
	return hex.EncodeToString(toByteArray(id))
}

//===

// getDatacenterId 获取数据中心 ID
func getDatacenterId(maxDatacenterId int64) int64 {
	macs := net.RawMacAddress()
	if len(macs) == 0 {
		panic("can not get mac")
	}
	mac := macs[0]
	macLen := len(mac)
	if macLen != 6 {
		panic("mac len must be 6")
	}

	low := int64(mac[macLen-2])
	high := int64(mac[macLen-1]) << 8
	id := ((0x000000FF & low) | (0x0000FF00 & high)) >> 6
	id = id % (maxDatacenterId + 1)
	return id
}

// getWorkerId 获取工作者 ID
func getWorkerId(datacenterId, maxWorkerId int64) int64 {
	mpid := strconv.FormatInt(datacenterId, 10) + strconv.Itoa(os.Getpid())
	hc := int64(hashCode(mpid))
	return (hc & 0xffff) % (maxWorkerId + 1)
}

// wait 休眠等待指定毫秒数
func wait(millis int64) {
	time.Sleep(time.Duration(millis) * time.Millisecond)
}

// tilNextMillis 自旋等待直到下一个毫秒
func tilNextMillis(lastTimestamp int64) int64 {
	timestamp := timeGen()
	for timestamp <= lastTimestamp {
		timestamp = timeGen()
	}
	return timestamp
}

// timeGen 返回当前时间的毫秒数
func timeGen() int64 {
	return time.Now().UnixMilli()
}

// nextInt 返回随机数, 范围 [startInclusive, endExclusive)
func nextInt(startInclusive int, endExclusive int) int {
	if startInclusive > endExclusive {
		panic("Start value must be smaller or equal to end value.")
	}
	if startInclusive < 0 {
		panic("Both range values must be non-negative.")
	}
	if startInclusive == endExclusive {
		return startInclusive
	}
	return startInclusive + localRand.Intn(endExclusive-startInclusive)
}

// toByteArray 将 int64 转换为字节数组
func toByteArray(value int64) []byte {
	return []byte{
		(byte)(value >> 56),
		(byte)(value >> 48),
		(byte)(value >> 40),
		(byte)(value >> 32),
		(byte)(value >> 24),
		(byte)(value >> 16),
		(byte)(value >> 8),
		(byte)(value),
	}
}

// hashCode 计算字符串的哈希值
func hashCode(value string) int {
	h := 0
	if len(value) > 0 {
		for _, r := range value {
			h = 31*h + int(r)
		}
	}
	return h
}

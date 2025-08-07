package snowflake

import (
	"encoding/hex"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBits(t *testing.T) {
	snowFlake1 := NewSnowFlake()
	datacenterId := getDatacenterId(maxDatacenterId)
	workerId := getWorkerId(datacenterId, maxWorkerId)
	snowFlake2 := NewSnowFlakeWithConf(datacenterId, workerId)
	assert.Equal(t, snowFlake1.GetDatacenterId(), snowFlake2.GetDatacenterId())
	assert.Equal(t, snowFlake1.GetWorkerId(), snowFlake2.GetWorkerId())
	assert.Greater(t, snowFlake1.GetDatacenterId(), int64(0))
	assert.Greater(t, snowFlake1.GetWorkerId(), int64(0))
}

func TestOrder(t *testing.T) {
	snowFlake := NewSnowFlake()
	var ids []int64
	for i := 0; i < (1 << 14); i++ {
		ids = append(ids, snowFlake.NextId())
	}
	assert.Equal(t, 1<<14, len(ids))
	//排序
	ids_copy := make([]int64, len(ids))
	copy(ids_copy, ids)
	sort.Slice(ids_copy, func(i, j int) bool {
		return ids_copy[i] < ids_copy[j]
	})
	assert.Equal(t, ids, ids_copy)
}

func TestConcurrent(t *testing.T) {
	const concurrent = 1 << 5
	const loop = 1 << 14
	snowFlake := NewSnowFlake()
	var ids []int64
	wg := sync.WaitGroup{}
	wg.Add(concurrent)
	lock := sync.Mutex{}
	for i := 0; i < concurrent; i++ {
		go func() {
			for j := 0; j < loop; j++ {
				lock.Lock()
				ids = append(ids, snowFlake.NextId())
				lock.Unlock()
			}
			wg.Done()
		}()
	}

	wg.Wait()
	assert.Equal(t, concurrent*loop, len(ids))

	//去重
	m := map[int64]bool{}
	for _, id := range ids {
		m[id] = true
	}
	assert.Equal(t, len(ids), len(m))
}

func TestNextIdHex(t *testing.T) {
	snowFlake := NewSnowFlake()
	idHex := snowFlake.NextIdHex()
	assert.Equal(t, 16, len(idHex))
}

func TestNextInt(t *testing.T) {
	const startInclusive = 1
	const endExclusive = 3
	for i := 0; i < 100; i++ {
		value := nextInt(startInclusive, endExclusive)
		assert.GreaterOrEqual(t, value, startInclusive)
		assert.Less(t, value, endExclusive)
	}
}

func TestToByteArrayLong(t *testing.T) {
	var value int64 = 0x0102030405060708
	assert.Equal(t, "0102030405060708", hex.EncodeToString(toByteArray(value)))
	value = 1953347643828977665
	assert.Equal(t, "1b1baf553145c001", hex.EncodeToString(toByteArray(value)))
}

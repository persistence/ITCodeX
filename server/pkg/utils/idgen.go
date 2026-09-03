package utils

import (
	"math/rand"
	"sync"
	"time"

	"github.com/gogf/gf/v2/util/grand"
	"github.com/matoous/go-nanoid/v2"
)

const (
	snowflakeEpoch     = int64(1704067200000)
	snowflakeMachineBits = uint(10)
	snowflakeSequenceBits = uint(12)
	snowflakeMaxMachineID = int64(-1) ^ (int64(-1) << snowflakeMachineBits)
	snowflakeMaxSequence  = int64(-1) ^ (int64(-1) << snowflakeSequenceBits)
	snowflakeTimeShift    = snowflakeMachineBits + snowflakeSequenceBits
	snowflakeMachineShift = snowflakeSequenceBits
)

type SnowflakeGenerator struct {
	mu        sync.Mutex
	machineID int64
	sequence  int64
	lastTime  int64
}

var (
	defaultGenerator *SnowflakeGenerator
	once             sync.Once
)

func init() {
	once.Do(func() {
		defaultGenerator = NewSnowflakeGenerator(rand.Int63n(snowflakeMaxMachineID + 1))
	})
}

func NewSnowflakeGenerator(machineID int64) *SnowflakeGenerator {
	if machineID < 0 || machineID > snowflakeMaxMachineID {
		machineID = rand.Int63n(snowflakeMaxMachineID + 1)
	}
	return &SnowflakeGenerator{
		machineID: machineID,
		lastTime:  -1,
	}
}

func (g *SnowflakeGenerator) NextID() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli()

	if now < g.lastTime {
		now = g.lastTime
	}

	if now == g.lastTime {
		g.sequence = (g.sequence + 1) & snowflakeMaxSequence
		if g.sequence == 0 {
			for now <= g.lastTime {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		g.sequence = 0
	}

	g.lastTime = now

	id := ((now - snowflakeEpoch) << snowflakeTimeShift) |
		(g.machineID << snowflakeMachineShift) |
		g.sequence

	return id
}

func SetMachineID(machineID int64) {
	once.Do(func() {
		defaultGenerator = NewSnowflakeGenerator(machineID)
	})
}

func NextID() int64 {
	return defaultGenerator.NextID()
}

func UUID() string {
	return grand.S(32)
}

func NanoID(size ...int) string {
	length := 21
	if len(size) > 0 && size[0] > 0 {
		length = size[0]
	}
	id, _ := gonanoid.New(length)
	return id
}

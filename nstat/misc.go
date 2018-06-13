package nstat

import (
	"sync"
)

// 存储最新消费日志的位置
var Los *LogOffsetSet = NewLogOffsetSet()

type LogOffsetSet struct {
	sync.RWMutex
	offsets map[string]string
}

func NewLogOffsetSet() *LogOffsetSet {
	return &LogOffsetSet{
		offsets: make(map[string]string),
	}
}

func (los *LogOffsetSet) Set(topic string, msgid string) {
	los.Lock()
	los.offsets[topic] = msgid
	los.Unlock()
}

func (los *LogOffsetSet) Map() map[string]string {
	los.RLock()
	mapset := make(map[string]string)
	for topic, msgid := range los.offsets {
		mapset[topic] = msgid
	}
	los.RUnlock()
	return mapset
}

func (los *LogOffsetSet) Clear() {
	los.Lock()
	los.offsets = make(map[string]string)
	los.Unlock()
}

// 存储消费失败的消息
var Lfs *LogFailedSet = NewLogFailedSet()

type LogFailedSet struct {
	sync.RWMutex
	idmap map[string]map[string]struct{}
}

func NewLogFailedSet() *LogFailedSet {
	return &LogFailedSet{
		idmap: make(map[string]map[string]struct{}),
	}
}

func (lfs *LogFailedSet) Add(topic string, msgid string) {
	lfs.Lock()
	if _, ok := lfs.idmap[topic]; ok {
		lfs.idmap[topic][msgid] = struct{}{}
	} else {
		lfs.idmap[topic] = make(map[string]struct{})
	}
	lfs.Unlock()
}

func (lfs *LogFailedSet) Map() map[string]map[string]struct{} {
	lfs.RLock()
	mapset := make(map[string]map[string]struct{})
	for topic, value := range lfs.idmap {
		mapset[topic] = make(map[string]struct{})
		for msgid, _ := range value {
			mapset[topic][msgid] = struct{}{}
		}
	}
	lfs.RUnlock()
	return mapset
}

func (lfs *LogFailedSet) Clear() {
	lfs.Lock()
	lfs.idmap = make(map[string]map[string]struct{})
	lfs.Unlock()
}

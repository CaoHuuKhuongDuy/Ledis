package storage

import (
	"context"
	"fmt"
	"github.com/petar/GoLLRB/llrb"
	"sync"
	"time"
)

const (
	cleanInterval = 1 * time.Second
)

type item struct {
	key *Key
}

func (i *item) Less(than llrb.Item) bool {
	if than.(*item).key == nil {
		return false
	}
	return i.key.getExpireTime().Before(than.(*item).key.getExpireTime())
}

type garbageCollector struct {
	tree   *llrb.LLRB
	ledis  *Ledis
	cancel context.CancelFunc
	mu     sync.RWMutex
}

func newGarbageCollector(ledis *Ledis) *garbageCollector {
	ctx, cancel := context.WithCancel(context.Background())
	gb := &garbageCollector{
		tree:   llrb.New(),
		ledis:  ledis,
		cancel: cancel,
	}
	go gb.clean(ctx)
	return gb
}

func (gc *garbageCollector) add(key *Key) {
	if key == nil {
		return
	}
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.tree.InsertNoReplace(&item{key: key})
}

func (gc *garbageCollector) remove(key *Key) {
	if key == nil {
		return
	}
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.tree.Delete(&item{key: key})
}

func (gc *garbageCollector) clean(ctx context.Context) {
	ticker := time.NewTicker(cleanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			gc.mu.RLock()
			now := time.Now()
			var removedKeys []*Key
			gc.tree.AscendGreaterOrEqual(&item{}, func(i llrb.Item) bool {
				if i.(*item).key.getExpireTime().After(now) {
					return false
				}
				removedKeys = append(removedKeys, i.(*item).key)
				return true
			})
			gc.mu.RUnlock()
			fmt.Println("[GB] cleaning", removedKeys)
			for _, removedKey := range removedKeys {
				_ = gc.ledis.deleteKey(removedKey)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (gc *garbageCollector) clone(ledis *Ledis) *garbageCollector {
	ctx, cancel := context.WithCancel(context.Background())
	newGC := &garbageCollector{
		tree:   llrb.New(),
		ledis:  ledis,
		cancel: cancel,
	}
	gc.tree.AscendGreaterOrEqual(&item{}, func(i llrb.Item) bool {
		newGC.add(i.(*item).key)
		return true
	})
	go newGC.clean(ctx)
	return newGC
}

func (gc *garbageCollector) stop() {
	gc.cancel()
}

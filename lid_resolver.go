package antiban

import (
	"container/list"
	"sync"
)

type LidResolver struct {
	mu       sync.RWMutex
	lidToPN  map[string]string
	pnToLID  map[string]string
	order    *list.List
	maxSize  int
}

func NewLidResolver(maxSize int) *LidResolver {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &LidResolver{
		lidToPN: make(map[string]string),
		pnToLID: make(map[string]string),
		order:   list.New(),
		maxSize: maxSize,
	}
}

func (lr *LidResolver) Learn(lid, pn string) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if oldPN, ok := lr.lidToPN[lid]; ok {
		delete(lr.pnToLID, oldPN)
		lr.removeFromOrder(lid)
	}
	if oldLID, ok := lr.pnToLID[pn]; ok {
		delete(lr.lidToPN, oldLID)
		lr.removeFromOrder(oldLID)
	}

	lr.lidToPN[lid] = pn
	lr.pnToLID[pn] = lid
	lr.order.PushFront(lid)

	if len(lr.lidToPN) > lr.maxSize {
		back := lr.order.Back()
		if back != nil {
			evictLID := back.Value.(string)
			if evictPN, ok := lr.lidToPN[evictLID]; ok {
				delete(lr.pnToLID, evictPN)
			}
			delete(lr.lidToPN, evictLID)
			lr.order.Remove(back)
		}
	}
}

func (lr *LidResolver) removeFromOrder(lid string) {
	for e := lr.order.Front(); e != nil; e = e.Next() {
		if e.Value.(string) == lid {
			lr.order.Remove(e)
			return
		}
	}
}

func (lr *LidResolver) ResolveCanonical(jid string) string {
	lr.mu.RLock()
	defer lr.mu.RUnlock()

	if pn, ok := lr.lidToPN[jid]; ok {
		return pn
	}
	if lid, ok := lr.pnToLID[jid]; ok {
		return lid
	}
	return jid
}

func (lr *LidResolver) GetLID(pn string) string {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	return lr.pnToLID[pn]
}

func (lr *LidResolver) GetPN(lid string) string {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	return lr.lidToPN[lid]
}

func (lr *LidResolver) HasMapping(jid string) bool {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	_, ok1 := lr.lidToPN[jid]
	_, ok2 := lr.pnToLID[jid]
	return ok1 || ok2
}

func (lr *LidResolver) GetMapping() map[string]string {
	lr.mu.RLock()
	defer lr.mu.RUnlock()
	result := make(map[string]string, len(lr.lidToPN))
	for k, v := range lr.lidToPN {
		result[k] = v
	}
	return result
}

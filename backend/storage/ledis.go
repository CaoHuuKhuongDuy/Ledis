package storage

import (
	"errors"
	"fmt"
	"ledis/utils"
	"sync"
	"time"
)

// enum for key type
type KeyType int

const (
	KeyTypeString KeyType = 1
	KeyTypeSet    KeyType = 2
)

type Key struct {
	name        string
	_type       KeyType
	lastRenewed time.Time
	expireTime  time.Duration
}

func (k *Key) renew() {
	k.lastRenewed = time.Now()
}

func (k *Key) getExpireTime() time.Time {
	return k.lastRenewed.Add(k.expireTime)
}

func (k *Key) isExpired() bool {
	return k.expireTime != 0 && k.getExpireTime().Before(time.Now())
}

func newKey(key string, keyType KeyType, expireTime time.Duration) *Key {
	return &Key{
		name:        key,
		_type:       keyType,
		lastRenewed: time.Now(),
		expireTime:  expireTime,
	}
}

type Ledis struct {
	atomicData map[string]*string
	setData    map[string]*map[string]bool
	keys       map[string]*Key
	mu         sync.RWMutex
	snapshots  *Ledis
	gc         *garbageCollector
}

func NewLedis() *Ledis {
	l := &Ledis{
		atomicData: make(map[string]*string),
		setData:    make(map[string]*map[string]bool),
		keys:       make(map[string]*Key),
	}
	l.gc = newGarbageCollector(l)
	return l
}

func (l *Ledis) HandleCommand(cmd string) ([]string, error) {
	cmdArg := utils.ParseCmd(cmd)
	fmt.Println("Command:", cmdArg)
	keys, err := utils.ValidateCmdAndGetKeys(cmdArg)
	if err != nil {
		return nil, err
	}
	fmt.Println("Keys:", keys)
	var keyEntries []*Key
	if len(keys) > 0 {
		keyEntries, err = l.convertKeys(keys...)
		if err != nil {
			return nil, err
		}
		for _, keyEntry := range keyEntries {
			if keyEntry != nil && keyEntry.isExpired() {
				_ = l.deleteKey(keyEntry)
				keyEntry = nil
			}
		}
	}
	for _, keyEntry := range keyEntries {
		fmt.Println("Key entry:", keyEntry)
	}
	switch cmdArg[0] {
	case "SET":
		if keyEntries[0] == nil {
			keyEntries[0] = newKey(keys[0], KeyTypeString, 0)
			l.keys[keys[0]] = keyEntries[0]
		}
		if err = l.set(keyEntries[0], cmdArg[2]); err != nil {
			return nil, err
		}
	case "GET":
		value, err := l.get(keyEntries[0])
		if err != nil {
			return nil, err
		}
		return []string{value}, nil
	case "SADD":
		if keyEntries[0] == nil {
			keyEntries[0] = newKey(keys[0], KeyTypeSet, 0)
			l.keys[keys[0]] = keyEntries[0]
		}
		if err = l.sadd(keyEntries[0], cmdArg[2:]...); err != nil {
			return nil, err
		}
	case "SREM":
		if err = l.srem(keyEntries[0], cmdArg[2:]...); err != nil {
			return nil, err
		}
	case "SMEMBERS":
		return l.smembers(keyEntries[0])
	case "SINTER":
		return l.sinter(keyEntries...)
	case "KEYS":
		return l.listKeys(), nil
	case "DEL":
		if err = l.deleteKey(keyEntries[0]); err != nil {
			return nil, err
		}
	case "EXPIRE":
		timeDuration, err := utils.Str2TimeDuration(cmdArg[2])
		if err != nil {
			return nil, fmt.Errorf("invalid duration: %v", err)
		}
		if err = l.setExpireKey(keyEntries[0], timeDuration); err != nil {
			return nil, err
		}
	case "TTL":
		ttl, err := l.getTTL(keyEntries[0])
		if err != nil {
			return nil, err
		}
		return []string{fmt.Sprintf("%d", ttl)}, nil
	case "SAVE":
		if err = l.save(); err != nil {
			return nil, err
		}
	case "RESTORE":
		if err = l.restore(); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (l *Ledis) deleteKey(key *Key) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if key == nil {
		return errors.New("key not found")
	}

	if key._type == KeyTypeString {
		delete(l.atomicData, key.name)
	} else {
		delete(l.setData, key.name)
	}
	l.gc.remove(key)
	delete(l.keys, key.name)
	return nil
}

func (l *Ledis) set(key *Key, value string) error {

	l.mu.Lock()
	l.atomicData[key.name] = &value
	if key == nil || key._type != KeyTypeString {
		l.mu.Unlock()
		return errors.New("key is not valid, this key may be a set key and does not support SET command")
	}
	l.mu.Unlock()

	l.renewKey(key)
	return nil
}

func (l *Ledis) get(key *Key) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if key == nil || key._type != KeyTypeString {
		return "", errors.New("key is not valid, this key may be a set key and does not support GET command")
	}

	return *l.atomicData[key.name], nil
}

func (l *Ledis) sadd(key *Key, values ...string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if key == nil || key._type != KeyTypeSet {
		return errors.New("key is not valid, this key may be a string key and does not support SADD command")
	}

	for _, value := range values {
		if l.setData[key.name] == nil {
			l.setData[key.name] = &map[string]bool{}
		}
		(*l.setData[key.name])[value] = true
	}
	return nil
}

func (l *Ledis) srem(key *Key, values ...string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if key == nil || key._type != KeyTypeSet {
		return errors.New("key is not valid, this key may be a string key and does not support SREM command")
	}

	for _, value := range values {
		if _, ok := (*l.setData[key.name])[value]; ok {
			delete(*l.setData[key.name], value)
		}
	}
	return nil
}

func (l *Ledis) smembers(key *Key) ([]string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if key == nil || key._type != KeyTypeSet {
		return nil, errors.New("key is not valid, this key may be a string key and does not support SMEMBERS command")
	}

	return setDataToStrings(l.setData[key.name]), nil
}

func (l *Ledis) sinter(keys ...*Key) ([]string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(keys) == 0 {
		return nil, errors.New("no keys provided")
	}
	for _, key := range keys {
		if key == nil || key._type != KeyTypeSet {
			return nil, errors.New("key is not valid, this key may be a string key and does not support SINTER command")
		}
	}

	idOfSetHasMinSize := -1
	var minSize int
	for index, key := range keys {
		if key._type == KeyTypeString {
			continue
		}
		if idOfSetHasMinSize == -1 {
			minSize = len(*l.setData[key.name])
			idOfSetHasMinSize = index
			continue
		}
		if len(*l.setData[key.name]) < minSize {
			minSize = len(*l.setData[key.name])
			idOfSetHasMinSize = index
		}
	}
	if idOfSetHasMinSize == -1 {
		return nil, errors.New("no set key found")
	}

	// candidatesIntersection is the elements of the set with the minimum size
	candidatesIntersection := make(map[string]bool)
	for value := range *l.setData[keys[idOfSetHasMinSize].name] {
		candidatesIntersection[value] = true
	}

	for index, key := range keys {
		if key._type == KeyTypeString || index == idOfSetHasMinSize {
			continue
		}
		for value := range candidatesIntersection {
			if _, ok := (*l.setData[key.name])[value]; !ok {
				delete(candidatesIntersection, value)
			}
		}
	}

	return setDataToStrings(&candidatesIntersection), nil
}

func (l *Ledis) listKeys() []string {
	l.mu.RLock()
	keys := make([]string, 0, len(l.keys))
	needRemoveKeys := make([]string, 0)
	for key, keyEntry := range l.keys {
		if keyEntry.isExpired() {
			needRemoveKeys = append(needRemoveKeys, key)
			continue
		}
		keys = append(keys, key)
	}
	l.mu.RUnlock()
	for _, key := range needRemoveKeys {
		_ = l.deleteKey(l.keys[key])
	}
	return keys
}

func (l *Ledis) setExpireKey(key *Key, expireTime time.Duration) error {
	l.mu.Lock()
	if key == nil {
		l.mu.Unlock()
		return errors.New("key not found")
	}
	key.expireTime = expireTime
	l.mu.Unlock()
	l.renewKey(key)
	return nil
}

func (l *Ledis) getTTL(key *Key) (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if key == nil {
		return 0, errors.New("key not found")
	}

	if key.expireTime == 0 {
		return 0, errors.New("key has no expiration time")
	}

	expiration := key.lastRenewed.Add(key.expireTime)
	ttl := time.Until(expiration)

	if ttl <= 0 {
		return 0, errors.New("key expired")
	}
	// Convert to seconds
	ttlSeconds := uint64(ttl.Seconds())
	return ttlSeconds, nil
}

// Save creates a deep copy of the current Ledis instance and stores it in snapshots.
func (l *Ledis) save() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	snapshot := &Ledis{}
	_ = snapshot.copy(l)
	l.snapshots = snapshot
	return nil
}

func (l *Ledis) restore() error {
	if err := l.copy(l.snapshots); err != nil {
		return fmt.Errorf("failed to restore: %v", err)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	// Reset the snapshot after restoring
	l.snapshots = nil
	return nil
}

func (l *Ledis) copy(other *Ledis) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if other == nil {
		return errors.New("nil copy source")
	}

	// Deep copy atomicData
	l.atomicData = make(map[string]*string)
	for key, value := range other.atomicData {
		if value != nil {
			copyValue := *value
			l.atomicData[key] = &copyValue
		} else {
			l.atomicData[key] = nil
		}
	}

	// Deep copy setData
	l.setData = make(map[string]*map[string]bool)
	for key, set := range other.setData {
		if set != nil {
			newSet := make(map[string]bool)
			for subKey, subValue := range *set {
				newSet[subKey] = subValue
			}
			l.setData[key] = &newSet
		} else {
			l.setData[key] = nil
		}
	}

	// Deep copy keys
	l.keys = make(map[string]*Key)
	for key, keyStruct := range other.keys {
		if keyStruct != nil {
			newKey := *keyStruct
			l.keys[key] = &newKey
		} else {
			l.keys[key] = nil
		}
	}

	// Deep copy garbage collector
	if l.gc != nil {
		// Stop the current garbage collector
		l.gc.stop()
	}
	l.gc = other.gc.clone(l)

	return nil
}

func (l *Ledis) convertKeys(keys ...string) ([]*Key, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(keys) == 0 {
		return nil, errors.New("no keys provided")
	}

	keyEntries := make([]*Key, 0, len(keys))
	for _, key := range keys {
		keyEntry, ok := l.keys[key]
		if !ok {
			keyEntry = nil
		}
		keyEntries = append(keyEntries, keyEntry)
	}
	return keyEntries, nil
}

func (l *Ledis) renewKey(key *Key) {
	if key == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.gc.remove(key)
	key.renew()
	if key.expireTime != 0 {
		l.gc.add(key)
	}
}

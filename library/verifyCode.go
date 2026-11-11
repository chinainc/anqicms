package library

import (
	"math/rand"
	"strconv"
	"sync"
	"time"
)

type verifyCode struct {
	Expire int64
	key    string
	code   string
}

type verifyCodeCache struct {
	mu   sync.Mutex
	list map[string]*verifyCode
	size int
}

var CodeCache verifyCodeCache

func (v *verifyCodeCache) Generate(key string) string {
	expire := time.Now().Unix() + 1800
	rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := strconv.Itoa(100000 + rd.Intn(900000))
	if tmpCode, ok := v.list[key]; ok {
		tmpCode.Expire = expire
		return tmpCode.code
	}
	v.mu.Lock()
	node := &verifyCode{
		Expire: expire,
		key:    key,
		code:   code,
	}
	v.list[key] = node
	v.size++

	v.mu.Unlock()

	return code
}

func (v *verifyCodeCache) Get(key string, clear bool) string {
	if _, ok := v.list[key]; !ok {
		//数据不存在
		return ""
	}
	if clear {
		defer delete(v.list, key)
	}

	return v.list[key].code
}

func (v *verifyCodeCache) GetByCode(code string, clear bool) (key string) {
	for k := range v.list {
		if v.list[k].code == code {
			key = k

			if clear {
				delete(v.list, key)
			}
			return
		}
	}

	return
}

func (v *verifyCodeCache) GC() {
	for {
		timestamp := time.Now().Unix()
		v.mu.Lock()
		for k, item := range v.list {
			if item.Expire < timestamp {
				delete(v.list, k)
				v.size--
			}
		}
		v.mu.Unlock()
		time.Sleep(5 * time.Second)
	}
}

// Verify 校验验证码（本地扩展：上游版本没有该方法）。
// clear=true 时校验通过即消费；false 时保留，输错可重试。
func (v *verifyCodeCache) Verify(key string, code string, clear bool) bool {
	item, ok := v.list[key]
	if !ok {
		return false
	}
	if item.Expire < time.Now().Unix() || item.code != code {
		return false
	}
	if clear {
		v.mu.Lock()
		delete(v.list, key)
		v.size--
		v.mu.Unlock()
	}
	return true
}

// Delete 主动删除某个 key 的验证码（本地扩展）
func (v *verifyCodeCache) Delete(key string) {
	v.mu.Lock()
	if _, ok := v.list[key]; ok {
		delete(v.list, key)
		v.size--
	}
	v.mu.Unlock()
}

func init() {
	CodeCache = verifyCodeCache{
		size: 0,
		list: map[string]*verifyCode{},
	}
	//执行回收
	go CodeCache.GC()
}

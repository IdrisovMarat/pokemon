package pokecache

import (
	"fmt"
	"sync"
	"time"
)

type cacheEntry struct {
	createdAt time.Time
	val       []byte
}

type Cache struct {
	mu       *sync.Mutex
	data     map[string]cacheEntry
	interval time.Duration
	stop     chan struct{}
}

type Pokemonmain struct {
	Name      string
	CreatedAt time.Time
}

type Pokedex struct {
	mu   *sync.Mutex
	data map[string]Pokemonmain
}

func (p *Pokedex) Add(pokemon string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.data[pokemon] = Pokemonmain{
		Name:      pokemon,
		CreatedAt: time.Now(),
	}
	// fmt.Println("ВСЕГО ПОКЕМОНОВ В POKEDEX", len(p.data))
}

func NewPokedex() *Pokedex {

	pokedex := &Pokedex{
		mu:   &sync.Mutex{},
		data: make(map[string]Pokemonmain),
	}
	return pokedex
}

func NewCache(interval time.Duration) *Cache {

	cache := &Cache{
		mu:       &sync.Mutex{},
		data:     make(map[string]cacheEntry),
		interval: interval,
		stop:     make(chan struct{}),
	}
	go cache.reapLoop()

	return cache
}

func (c *Cache) reap() {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-c.interval)
	for key, entry := range c.data {
		if entry.createdAt.Before(cutoff) {
			delete(c.data, key)
		}

	}
}

func (c *Cache) reapLoop() {
	ticker := time.NewTicker(c.interval)

	for {
		select {
		case <-ticker.C:
			c.reap()
		case <-c.stop:
			return
		}
	}
}

func (c *Cache) Stop() {
	for key := range c.data {
		delete(c.data, key)
	}
	fmt.Println("...ALL CACHED DATA DELETED...")
	close(c.stop)
}

// Методы для работы с кэшем
func (c *Cache) Add(key string, val []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = cacheEntry{
		createdAt: time.Now(),
		val:       val,
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, exists := c.data[key]
	if !exists {
		return nil, false
	}
	return entry.val, true
}

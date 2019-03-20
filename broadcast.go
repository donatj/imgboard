package main

import "sync"

type imgdata []byte

type broadcast struct {
	channels map[chan imgdata]bool

	sync.Mutex
}

func newBroadcast() *broadcast {
	return &broadcast{
		channels: make(map[chan imgdata]bool),
	}
}

func (b *broadcast) Register() chan imgdata {
	b.Lock()
	defer b.Unlock()

	c := make(chan imgdata, 10)
	b.channels[c] = true

	return c
}

func (b *broadcast) Clear(c chan imgdata) {
	b.Lock()
	defer b.Unlock()

	delete(b.channels, c)
	close(c)
}

func (b *broadcast) Broadcast(data []byte) {
	b.Lock()
	defer b.Unlock()

	for c := range b.channels {
		c <- data
	}
}

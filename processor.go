package sse

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
)

var (
	ErrStreamerStopped = errors.New("this streamer stopped")
)

// Streamer SSE Streamer
type Streamer struct {
	userMap   map[string]map[ID]*Client
	clientMap map[ID]*Client
	options   Options
	stopped   bool
	mutex     sync.RWMutex
	wg        sync.WaitGroup
}

type Client struct {
	userKey      string
	connectionID ID
	send         chan *payload
}

type payload struct {
	event string
	data  []string
}

func NewStreamer(options Options) *Streamer {
	streamer := &Streamer{
		userMap:   map[string]map[ID]*Client{},
		clientMap: map[ID]*Client{},
		options:   options,
		stopped:   true,
	}
	return streamer
}

func (s *Streamer) Start() {
	s.mutex.Lock()
	s.stopped = false
	s.mutex.Unlock()
}

func (s *Streamer) Stop() {
	s.mutex.Lock()
	s.stopped = true
	s.mutex.Unlock()

	s.wg.Wait()

	s.mutex.Lock()
	for _, v := range s.userMap {
		for _, c := range v {
			close(c.send)
		}
	}
	s.userMap = map[string]map[ID]*Client{}
	s.mutex.Unlock()
}

func (s *Streamer) BroadcastJson(event string, data interface{}) error {
	json, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.Broadcast(event, string(json))
}

func (s *Streamer) MulticastJson(event string, data interface{}, users []string) error {
	json, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.Multicast(event, string(json), users)
}

func (s *Streamer) UnicastJson(event string, data interface{}, clientID ID) error {
	json, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.Unicast(event, string(json), clientID)
}

func (s *Streamer) Broadcast(event string, data string) error {
	targets := make([]*Client, 0)

	s.mutex.RLock()
	if s.stopped {
		s.mutex.RUnlock()
		return ErrStreamerStopped
	}

	for _, v := range s.userMap {
		for _, c := range v {
			targets = append(targets, c)
		}
	}
	s.mutex.RUnlock()

	s.send(targets, makePayload(event, data))
	return nil
}

func (s *Streamer) Multicast(event string, data string, users []string) error {
	targets := make([]*Client, 0)

	s.mutex.RLock()
	if s.stopped {
		s.mutex.RUnlock()
		return ErrStreamerStopped
	}

	done := make(map[string]bool, len(users))
	for _, v := range users {
		if ok := done[v]; ok {
			continue
		}
		for _, c := range s.userMap[v] {
			targets = append(targets, c)
		}
		done[v] = true
	}
	s.mutex.RUnlock()

	s.send(targets, makePayload(event, data))
	return nil
}

func (s *Streamer) Unicast(event string, data string, clientID ID) error {
	targets := make([]*Client, 1)

	s.mutex.RLock()
	if s.stopped {
		s.mutex.RUnlock()
		return ErrStreamerStopped
	}

	c, ok := s.clientMap[clientID]
	if ok {
		targets[0] = c
	}
	s.mutex.RUnlock()

	s.send(targets, makePayload(event, data))
	return nil
}

func (s *Streamer) NewClient(userKey string) (*Client, error) {
	client := &Client{
		userKey:      userKey,
		connectionID: generateRandomID(),
		send:         make(chan *payload, s.options.SendChanBufferSize),
	}
	if err := s.addClient(client); err != nil {
		close(client.send)
		return nil, err
	}
	return client, nil
}

func (s *Streamer) addClient(c *Client) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.stopped {
		return ErrStreamerStopped
	}

	if arr, ok := s.userMap[c.userKey]; ok {
		arr[c.connectionID] = c
	} else {
		new := make(map[ID]*Client)
		new[c.connectionID] = c
		s.userMap[c.userKey] = new
	}
	s.clientMap[c.connectionID] = c
	return nil
}

func (s *Streamer) removeClient(c *Client) error {
	s.mutex.Lock()
	arr, _ := s.userMap[c.userKey]
	delete(arr, c.connectionID)
	delete(s.clientMap, c.connectionID)
	s.mutex.Unlock()
	return nil
}

func (s *Streamer) send(targets []*Client, payload *payload) {
	s.wg.Add(1)
	for _, c := range targets {
		c.send <- payload
	}
	s.wg.Done()
}

func (c *Client) GetUserKey() string {
	return c.userKey
}

func (c *Client) GetConnectionID() ID {
	return c.connectionID
}

func makePayload(event string, data string) *payload {
	return &payload{
		event: event,
		data:  strings.SplitN(data, "\n", -1),
	}
}

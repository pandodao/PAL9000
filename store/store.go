package store

import (
	"sync"

	"github.com/pandodao/botastic-go"
)

type Store interface {
	GetConversationByKey(key string) (*botastic.Conversation, error)
	SetConversation(key string, conv *botastic.Conversation) error
}

type MemoryStore struct {
	convLock sync.Mutex
	convMap  map[string]*botastic.Conversation
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		convMap: make(map[string]*botastic.Conversation),
	}
}

func (s *MemoryStore) GetConversationByKey(key string) (*botastic.Conversation, error) {
	s.convLock.Lock()
	defer s.convLock.Unlock()

	return s.convMap[key], nil
}

func (s *MemoryStore) SetConversation(key string, conv *botastic.Conversation) error {
	s.convLock.Lock()
	defer s.convLock.Unlock()

	s.convMap[key] = conv
	return nil
}

package client

import (
	"github.com/google/uuid"
	"github.com/tomasdemarco/iso8583/message"
	"sync"
	"time"
)

type OngoingTransactions struct {
	List map[string]OngoingTransaction
	mu   *sync.RWMutex
}

type OngoingTransaction struct {
	RequestId *uuid.UUID
	StartTime time.Time
	Message   chan message.Message
}

func NewOngoingTransactions() *OngoingTransactions {
	return &OngoingTransactions{
		List: make(map[string]OngoingTransaction),
		mu:   &sync.RWMutex{},
	}
}

func (s *OngoingTransactions) Add(key string, id *uuid.UUID) chan message.Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	msgChan := make(chan message.Message, 1)

	transaction := OngoingTransaction{id, now, msgChan}

	s.List[key] = transaction

	return msgChan
}

func (s *OngoingTransactions) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.List, id)
}

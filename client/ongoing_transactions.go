package client

import (
	"github.com/tomasdemarco/go-pos/context"
	"github.com/tomasdemarco/iso8583/message"
	"sync"
)

type OngoingTransactions struct {
	List map[string]OngoingTransaction
	mu   *sync.RWMutex
}

type OngoingTransaction struct {
	Ctx     *context.RequestContext
	Message chan message.Message
}

func NewOngoingTransactions() *OngoingTransactions {
	return &OngoingTransactions{
		List: make(map[string]OngoingTransaction),
		mu:   &sync.RWMutex{},
	}
}

func (s *OngoingTransactions) Add(ctx *context.RequestContext, key string) chan message.Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgChan := make(chan message.Message, 1)

	transaction := OngoingTransaction{ctx, msgChan}

	s.List[key] = transaction

	return msgChan
}

func (s *OngoingTransactions) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.List, id)
}

func (s *OngoingTransactions) IsChanClosed(id string) bool {
	select {
	case <-s.List[id].Message:
		return true
	default:
	}

	return false
}

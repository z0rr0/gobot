package serve

import (
	"context"
	"sync"

	"github.com/z0rr0/gobot/cmd"
)

// HandlerType is a type for command handler.
type HandlerType func(context.Context, *cmd.Event) error

// SyncCommands is a global map of chats which should be locked during command execution.
type SyncCommands struct {
	sync.RWMutex
	commands map[string]struct{}    // commands which should be locked
	chats    map[string]*sync.Mutex // chats mutexes
}

func NewSyncCommands(commands []string) *SyncCommands {
	sc := &SyncCommands{
		commands: make(map[string]struct{}, len(commands)),
		chats:    make(map[string]*sync.Mutex),
	}

	for _, command := range commands {
		sc.commands[command] = struct{}{}
	}

	return sc
}

// initChat initializes chat mutex if it doesn't exist.
func (s *SyncCommands) chatMutex(command, chat string) *sync.Mutex {
	s.RLock()
	_, ok := s.commands[command]
	if !ok {
		s.RUnlock()
		return nil
	}

	mu, ok := s.chats[chat]
	if ok {
		s.RUnlock()
		return mu
	}

	s.RUnlock()
	// no mutex for chat, but it's required, so create it
	s.Lock()
	defer s.Unlock()

	if mu, ok = s.chats[chat]; ok {
		// if between RUnlock and Lock another goroutine created mutex
		return mu
	}

	mu = &sync.Mutex{}
	s.chats[chat] = mu
	return mu
}

// Decorate returns a decorated function for the chat.
func (s *SyncCommands) Decorate(command, chat string, f HandlerType) HandlerType {
	mu := s.chatMutex(command, chat)
	if mu == nil {
		return f
	}

	// execute f with lock
	return func(ctx context.Context, e *cmd.Event) error {
		mu.Lock()
		defer mu.Unlock()

		return f(ctx, e)
	}
}

package serve

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/z0rr0/gobot/cmd"
)

func TestNewSyncCommands(t *testing.T) {
	const lockedCommand = "/test"
	var (
		timeout      = 10 * time.Millisecond
		totalTimeout = 58 * time.Millisecond
	)

	testCases := []struct {
		name     string
		workers  int
		expected uint64
		commands []string
		chats    []string
	}{
		{
			name:     "no_locked_commands",
			workers:  5,
			expected: 30,
			commands: []string{},
			chats:    []string{"chat", "chat", "chat", "chat", "chat"},
		},
		{
			name:     "free_commands",
			workers:  5,
			expected: 30,
			commands: []string{"/go", "/shuffle"},
			chats:    []string{"chat", "chat", "chat", "chat", "chat"},
		},
		{
			name:     "with_lock",
			workers:  5,
			expected: 6,
			commands: []string{lockedCommand},
			chats:    []string{"chat", "chat", "chat", "chat", "chat"},
		},
		{
			name:     "mixed_commands",
			workers:  5,
			expected: 6,
			commands: []string{lockedCommand, "/go"},
			chats:    []string{"chat", "chat", "chat", "chat", "chat"},
		},
		{
			name:     "diff_chats",
			workers:  5,
			expected: 30,
			commands: []string{lockedCommand},
			chats:    []string{"chat1", "chat2", "chat3", "chat4", "chat5"},
		},
		{
			name:     "mixed_chats",
			workers:  5,
			expected: 18, // 6 (3 "chat") + 6 + 6
			commands: []string{lockedCommand, "/go"},
			chats:    []string{"chat", "chat", "chat", "chat1", "chat2"},
		},
		{
			name:     "mixed_chats_pairs",
			workers:  5,
			expected: 12,
			commands: []string{lockedCommand, "/go"},
			chats:    []string{"chat1", "chat1", "chat1", "chat2", "chat2"},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			var (
				result atomic.Uint64
				wg     sync.WaitGroup
				sc     = NewSyncCommands(tc.commands)
			)

			wg.Add(tc.workers)
			ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
			defer cancel()

			for j := 0; j < tc.workers; j++ {
				handler := func(c context.Context, _ *cmd.Event) error {
					for e := c.Err(); e == nil; e = c.Err() {
						result.Add(1)
						time.Sleep(timeout)
					}
					return nil
				}
				handler = sc.Decorate(lockedCommand, tc.chats[j], handler)
				go func() {
					_ = handler(ctx, nil)
					wg.Done()
				}()
			}

			wg.Wait()
			if got := result.Load(); got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

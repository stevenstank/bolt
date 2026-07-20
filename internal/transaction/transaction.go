package transaction

import (
	"github.com/stevenstank/bolt/internal/protocol"
)

// Transaction represents a queued transaction.
type Transaction struct {
	queued  []protocol.Command
	inMulti bool
}

// New creates a new transaction.
func New() *Transaction {
	return &Transaction{
		queued: make([]protocol.Command, 0),
	}
}

// InMulti returns whether the transaction is in MULTI mode.
func (t *Transaction) InMulti() bool {
	return t.inMulti
}

// StartMulti begins a transaction.
func (t *Transaction) StartMulti() {
	t.inMulti = true
	t.queued = make([]protocol.Command, 0)
}

// Queue adds a command to the transaction queue.
func (t *Transaction) Queue(cmd protocol.Command) {
	t.queued = append(t.queued, cmd)
}

// Exec returns the queued commands and clears the queue.
func (t *Transaction) Exec() []protocol.Command {
	commands := t.queued
	t.queued = make([]protocol.Command, 0)
	t.inMulti = false
	return commands
}

// Discard clears the queued commands and exits MULTI mode.
func (t *Transaction) Discard() {
	t.queued = make([]protocol.Command, 0)
	t.inMulti = false
}

// QueuedCount returns the number of queued commands.
func (t *Transaction) QueuedCount() int {
	return len(t.queued)
}

package command

import (
	"fmt"

	"github.com/stevenstank/bolt/internal/protocol"
)

// Processor parses client lines and dispatches commands.
type Processor struct {
	dispatcher *Dispatcher
}

// NewProcessor creates a Processor backed by dispatcher.
func NewProcessor(dispatcher *Dispatcher) *Processor {
	return &Processor{dispatcher: dispatcher}
}

// Process parses and executes one plain-text command line.
func (p *Processor) Process(line string) string {
	cmd, err := protocol.Parse(line)
	if err != nil {
		return fmt.Sprintf("ERR %v", err)
	}
	return p.dispatcher.Dispatch(cmd)
}

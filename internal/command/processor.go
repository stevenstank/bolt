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

// NewProcessorWithEngine creates a Processor with a fresh transaction for per-client isolation.
func NewProcessorWithEngine(engine storageEngine) *Processor {
	return &Processor{dispatcher: NewDispatcher(engine)}
}

// Process parses and executes one plain-text command line.
func (p *Processor) Process(line string) string {
	cmd, err := protocol.Parse(line)
	if err != nil {
		return fmt.Sprintf("ERR %v", err)
	}
	return p.dispatcher.Dispatch(cmd)
}

// Clone creates a new processor with a fresh transaction for per-client isolation.
func (p *Processor) Clone() interface{} {
	clone := NewProcessorWithEngine(p.dispatcher.engine)
	clone.dispatcher.SetInfo(p.dispatcher.info)
	return clone
}

// SetInfo sets the info provider for this processor.
func (p *Processor) SetInfo(info InfoProvider) {
	p.dispatcher.SetInfo(info)
}

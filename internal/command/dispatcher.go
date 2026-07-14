package command

import (
	"fmt"

	"github.com/stevenstank/bolt/internal/protocol"
)

type engine interface {
	Set(key, value string) error
	Get(key string) (string, bool)
}

// Dispatcher validates commands and routes them to storage.
type Dispatcher struct {
	engine engine
}

// NewDispatcher creates a command dispatcher.
func NewDispatcher(engine engine) *Dispatcher {
	return &Dispatcher{engine: engine}
}

// Dispatch executes a parsed command and returns a plain-text response.
func (d *Dispatcher) Dispatch(cmd protocol.Command) string {
	switch cmd.Name {
	case "SET":
		if len(cmd.Args) != 2 {
			return "ERR SET requires key and value"
		}
		if err := d.engine.Set(cmd.Args[0], cmd.Args[1]); err != nil {
			return fmt.Sprintf("ERR %v", err)
		}
		return "OK"
	case "GET":
		if len(cmd.Args) != 1 {
			return "ERR GET requires key"
		}
		value, ok := d.engine.Get(cmd.Args[0])
		if !ok {
			return "(nil)"
		}
		return value
	default:
		return fmt.Sprintf("ERR unknown command %q", cmd.Name)
	}
}

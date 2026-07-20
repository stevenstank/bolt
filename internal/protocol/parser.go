package protocol

import (
	"errors"
	"strings"
)

// Command is a parsed plain-text client command.
type Command struct {
	Name string
	Args []string
}

// Parse converts one newline-delimited client line into a command.
func Parse(line string) (Command, error) {
	trimmed := strings.TrimSpace(line)
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return Command{}, errors.New("empty command")
	}

	name := strings.ToUpper(fields[0])
	if name == "SET" {
		return parseSet(trimmed, fields[0]), nil
	}

	return Command{
		Name: name,
		Args: fields[1:],
	}, nil
}

func parseSet(line, commandName string) Command {
	rest := strings.TrimSpace(line[len(commandName):])
	if rest == "" {
		return Command{Name: "SET"}
	}

	keyFields := strings.Fields(rest)
	key := keyFields[0]
	if len(keyFields) >= 4 && strings.EqualFold(keyFields[len(keyFields)-2], "EX") {
		value := strings.TrimSpace(strings.Join(keyFields[1:len(keyFields)-2], " "))
		if value == "" {
			return Command{Name: "SET", Args: []string{key}}
		}
		return Command{Name: "SET", Args: []string{key, value, "EX", keyFields[len(keyFields)-1]}}
	}

	value := strings.TrimSpace(rest[len(key):])
	if value == "" {
		return Command{
			Name: "SET",
			Args: []string{key},
		}
	}

	return Command{
		Name: "SET",
		Args: []string{key, value},
	}
}

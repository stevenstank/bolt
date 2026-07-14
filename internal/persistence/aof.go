package persistence

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// AOF stores write operations in an append-only log.
type AOF struct {
	path string
}

// NewAOF creates an append-only file handle for path.
func NewAOF(path string) *AOF {
	return &AOF{path: path}
}

// AppendSet records a SET operation.
func (a *AOF) AppendSet(key, value string) error {
	file, err := os.OpenFile(a.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprint(file, formatSetRecord(key, value))
	return err
}

// Load replays the append-only file and returns the resulting data.
func (a *AOF) Load() (map[string]string, error) {
	file, err := os.OpenFile(a.path, os.O_RDWR, 0o644)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	data := map[string]string{}
	reader := bufio.NewReader(file)
	var offset int64
	for {
		line, err := reader.ReadString('\n')
		if errors.Is(err, io.EOF) && line == "" {
			break
		}
		if errors.Is(err, io.EOF) {
			if truncateErr := file.Truncate(offset); truncateErr != nil {
				return nil, truncateErr
			}
			break
		}
		if err != nil {
			return nil, err
		}

		offset += int64(len(line))
		line = strings.TrimSuffix(line, "\n")
		key, value, err := parseSetRecord(line)
		if err != nil {
			return nil, err
		}
		data[key] = value
	}
	return data, nil
}

func formatSetRecord(key, value string) string {
	return fmt.Sprintf("SET\t%d:%s\t%d:%s\n", len(key), key, len(value), value)
}

func parseSetRecord(line string) (string, string, error) {
	parts := strings.Split(line, "\t")
	if len(parts) != 3 || parts[0] != "SET" {
		return "", "", fmt.Errorf("invalid AOF record: %q", line)
	}

	key, err := parseLengthPrefixedField(parts[1])
	if err != nil {
		return "", "", err
	}
	value, err := parseLengthPrefixedField(parts[2])
	if err != nil {
		return "", "", err
	}
	return key, value, nil
}

func parseLengthPrefixedField(field string) (string, error) {
	lengthText, value, ok := strings.Cut(field, ":")
	if !ok {
		return "", fmt.Errorf("invalid AOF field: %q", field)
	}

	length, err := strconv.Atoi(lengthText)
	if err != nil {
		return "", fmt.Errorf("invalid AOF field length %q: %w", lengthText, err)
	}
	if length != len(value) {
		return "", fmt.Errorf("invalid AOF field length: expected %d bytes, got %d", length, len(value))
	}
	return value, nil
}

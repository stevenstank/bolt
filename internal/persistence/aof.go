package persistence

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/stevenstank/bolt/internal/record"
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
func (a *AOF) AppendSet(key, value string, expiresAt time.Time) error {
	file, err := os.OpenFile(a.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprint(file, formatSetRecord(key, value, expiresAt))
	return err
}

// Load replays the append-only file and returns the resulting data.
func (a *AOF) Load() (map[string]record.Entry, error) {
	file, err := os.OpenFile(a.path, os.O_RDWR, 0o644)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]record.Entry{}, nil
		}
		return nil, err
	}
	defer file.Close()

	data := map[string]record.Entry{}
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
		key, entry, err := parseSetRecord(line)
		if err != nil {
			return nil, err
		}
		if entry.Expired(time.Now()) {
			continue
		}
		data[key] = entry
	}
	return data, nil
}

func formatSetRecord(key, value string, expiresAt time.Time) string {
	if expiresAt.IsZero() {
		return fmt.Sprintf("SET\t%d:%s\t%d:%s\n", len(key), key, len(value), value)
	}

	expiresAtText := strconv.FormatInt(expiresAt.UnixNano(), 10)
	return fmt.Sprintf("SET\t%d:%s\t%d:%s\t%d:%s\n", len(key), key, len(expiresAtText), expiresAtText, len(value), value)
}

func parseSetRecord(line string) (string, record.Entry, error) {
	parts := strings.Split(line, "\t")
	if parts[0] != "SET" || (len(parts) != 3 && len(parts) != 4) {
		return "", record.Entry{}, fmt.Errorf("invalid AOF record: %q", line)
	}

	key, err := parseLengthPrefixedField(parts[1])
	if err != nil {
		return "", record.Entry{}, err
	}

	var expiresAt time.Time
	var valueField string
	if len(parts) == 3 {
		valueField = parts[2]
	} else {
		expiresAtText, err := parseLengthPrefixedField(parts[2])
		if err != nil {
			return "", record.Entry{}, err
		}
		expiresAtUnixNano, err := strconv.ParseInt(expiresAtText, 10, 64)
		if err != nil {
			return "", record.Entry{}, fmt.Errorf("invalid AOF expiry %q: %w", expiresAtText, err)
		}
		expiresAt = time.Unix(0, expiresAtUnixNano)
		valueField = parts[3]
	}

	value, err := parseLengthPrefixedField(valueField)
	if err != nil {
		return "", record.Entry{}, err
	}
	return key, record.Entry{Value: value, ExpiresAt: expiresAt}, nil
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

package persistence

import (
	"bufio"
	"os"
	"sort"
	"time"

	"github.com/stevenstank/bolt/internal/record"
)

// SaveSnapshot writes a deterministic point-in-time copy of data.
func SaveSnapshot(path string, data map[string]record.Entry) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	now := time.Now()
	for _, key := range keys {
		value := data[key]
		if value.Expired(now) {
			continue
		}
		if _, err := file.WriteString(formatSetRecord(key, value.Value, value.ExpiresAt)); err != nil {
			return err
		}
	}
	return nil
}

// LoadSnapshot reads a snapshot file and returns its data.
func LoadSnapshot(path string) (map[string]record.Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]record.Entry{}, nil
		}
		return nil, err
	}
	defer file.Close()

	data := map[string]record.Entry{}
	now := time.Now()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, err := parseSetRecord(scanner.Text())
		if err != nil {
			return nil, err
		}
		if value.Expired(now) {
			continue
		}
		data[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return data, nil
}
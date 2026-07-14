package persistence

import (
	"bufio"
	"os"
	"sort"
)

// SaveSnapshot writes a deterministic point-in-time copy of data.
func SaveSnapshot(path string, data map[string]string) error {
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

	for _, key := range keys {
		value := data[key]
		if _, err := file.WriteString(formatSetRecord(key, value)); err != nil {
			return err
		}
	}
	return nil
}

// LoadSnapshot reads a snapshot file and returns its data.
func LoadSnapshot(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	data := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, err := parseSetRecord(scanner.Text())
		if err != nil {
			return nil, err
		}
		data[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return data, nil
}

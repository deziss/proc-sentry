package internal

import (
	"log"
	"os"
	"strings"
)

// LoadUserMap reads /etc/passwd and returns a uid->username map.
func LoadUserMap(path string) map[string]string {
	m := make(map[string]string)

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Warning: Failed to read %s: %v", path, err)
		return m
	}

	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			m[parts[2]] = parts[0]
		}
	}
	log.Printf("Loaded %d users from %s", len(m), path)
	return m
}

// ResolveUser returns the username for a UID, falling back to the numeric UID.
func ResolveUser(userMap map[string]string, uid string) string {
	if name, ok := userMap[uid]; ok {
		return name
	}
	return uid
}

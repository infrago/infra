package infra

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

const defaultTokenIDHexLength = 16

// GenerateTokenID returns a random hex token id with configurable length.
// Default length is 16.
func GenerateTokenID(lengths ...int) string {
	length := defaultTokenIDHexLength
	if len(lengths) > 0 {
		length = normalizeTokenIDLength(lengths[0])
	}
	return generateTokenIDByLength(length)
}

func generateTokenIDByLength(length int) string {
	length = normalizeTokenIDLength(length)
	raw := make([]byte, (length+1)/2)
	if _, err := rand.Read(raw); err != nil {
		return normalizeGeneratedTokenID(Generate(), length)
	}
	id := hex.EncodeToString(raw)
	if len(id) > length {
		id = id[:length]
	}
	if len(id) < length {
		id += strings.Repeat("1", length-len(id))
	}
	// Keep first char non-zero for prettier visual ids.
	if id != "" && id[0] == '0' {
		id = "1" + id[1:]
	}
	return id
}

func normalizeGeneratedTokenID(id string, length int) string {
	length = normalizeTokenIDLength(length)
	id = strings.ToLower(strings.TrimSpace(id))
	if strings.HasPrefix(id, "-") {
		id = id[1:]
	}
	filtered := make([]byte, 0, len(id))
	for i := 0; i < len(id); i++ {
		c := id[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			filtered = append(filtered, c)
		}
	}
	id = string(filtered)
	if id == "" {
		return strings.Repeat("1", length)
	}
	if len(id) > length {
		id = id[len(id)-length:]
	} else if len(id) < length {
		id = strings.Repeat("1", length-len(id)) + id
	}
	if id[0] == '0' {
		id = "1" + id[1:]
	}
	return id
}

func normalizeTokenIDLength(length int) int {
	if length <= 0 {
		return defaultTokenIDHexLength
	}
	if length > 128 {
		return 128
	}
	return length
}

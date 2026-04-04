package sdk

import (
	"encoding/binary"

	"fiatjaf.com/nostr"
)

const eventAccessTimePrefix = byte('a')

// makeEventAccessTimeKey creates a key for storing event access time information.
// It uses the first 8 bytes of the event ID to create a compact key.
func makeEventAccessTimeKey(id nostr.ID) []byte {
	// format: 'a' + first 8 bytes of event ID
	key := make([]byte, 9)
	key[0] = eventAccessTimePrefix
	copy(key[1:], id[:8])
	return key
}

// encodeEventAccessTime serializes an EventAccessTime into a binary format.
func encodeEventAccessTime(t nostr.Timestamp) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf[0:8], uint64(t))
	return buf
}

// decodeEventAccessTime deserializes a binary-encoded EventAccessTime.
func decodeEventAccessTime(data []byte) nostr.Timestamp {
	return nostr.Timestamp(binary.BigEndian.Uint64(data[0:8]))
}

// trackEventAccessTime records the access time for an event.
func (sys *System) TrackEventAccessTime(id nostr.ID) {
	key := makeEventAccessTimeKey(id)
	sys.KVStore.Update(key, func(data []byte) ([]byte, error) {
		return encodeEventAccessTime(nostr.Now()), nil
	})
}

// GetEventAccessTime returns the access times for an event.
func (sys *System) GetEventAccessTime(id nostr.ID) nostr.Timestamp {
	key := makeEventAccessTimeKey(id)

	data, _ := sys.KVStore.Get(key)
	if data == nil {
		return 0
	}

	return decodeEventAccessTime(data)
}

func (sys *System) EraseAccessTime(id nostr.ID) error {
	key := makeEventAccessTimeKey(id)
	return sys.KVStore.Delete(key)
}

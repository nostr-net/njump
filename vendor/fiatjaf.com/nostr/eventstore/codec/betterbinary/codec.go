package betterbinary

import (
	"encoding/binary"
	"fmt"
	"math"

	"fiatjaf.com/nostr"
)

const (
	MaxKind         = math.MaxUint16
	MaxCreatedAt    = math.MaxUint32
	MaxContentSize  = math.MaxUint16
	MaxTagCount     = math.MaxUint16
	MaxTagItemCount = math.MaxUint8
	MaxTagItemSize  = math.MaxUint16
)

func Measure(evt nostr.Event) int {
	n := 135 // static base

	n += 2 + // tag section length
		2 + // number of tags
		len(evt.Tags)*3 // each tag offset + each tag item count
	for _, tag := range evt.Tags {
		n += len(tag) * 2 // item length for each item in this tag
		for _, item := range tag {
			n += len(item) // actual tag item
		}
	}

	// content length and actual content
	n += 2 + len(evt.Content)

	return n
}

func Marshal(evt nostr.Event, buf []byte) error {
	buf[0] = 0

	if evt.Kind > MaxKind {
		return fmt.Errorf("kind is too big: %d, max is %d", evt.Kind, uint16(MaxKind))
	}
	binary.LittleEndian.PutUint16(buf[1:3], uint16(evt.Kind))

	if evt.CreatedAt > MaxCreatedAt {
		return fmt.Errorf("created_at is too big: %d, max is %d", evt.CreatedAt, uint32(MaxCreatedAt))
	}
	binary.LittleEndian.PutUint32(buf[3:7], uint32(evt.CreatedAt))

	copy(buf[7:39], evt.ID[:])
	copy(buf[39:71], evt.PubKey[:])
	copy(buf[71:135], evt.Sig[:])

	tagBase := 135
	// buf[135:137] (tagsSectionLength) will be set later when we know the absolute size of the tags section

	ntags := len(evt.Tags)
	if ntags > MaxTagCount {
		return fmt.Errorf("can't encode too many tags: %d, max is %d", ntags, uint16(MaxTagCount))
	}
	binary.LittleEndian.PutUint16(buf[137:139], uint16(ntags))

	tagOffset := 2 + 2 + ntags*2
	for t, tag := range evt.Tags {
		binary.LittleEndian.PutUint16(buf[tagBase+2+2+t*2:], uint16(tagOffset))

		itemCount := len(tag)
		if itemCount > MaxTagItemCount {
			return fmt.Errorf("can't encode a tag with so many items: %d, max is %d", itemCount, uint8(MaxTagItemCount))
		}
		buf[tagBase+tagOffset] = uint8(itemCount)

		itemOffset := 1
		for _, item := range tag {
			itemSize := len(item)
			if itemSize > MaxTagItemSize {
				return fmt.Errorf("tag item is too large: %d, max is %d", itemSize, uint16(MaxTagItemSize))
			}

			binary.LittleEndian.PutUint16(buf[tagBase+tagOffset+itemOffset:], uint16(itemSize))
			copy(buf[tagBase+tagOffset+itemOffset+2:], []byte(item))
			itemOffset += 2 + len(item)
		}
		tagOffset += itemOffset
	}

	tagsSectionLength := tagOffset
	binary.LittleEndian.PutUint16(buf[tagBase:], uint16(tagsSectionLength))

	// content
	if contentLength := len(evt.Content); contentLength > MaxContentSize {
		return fmt.Errorf("content is too large: %d, max is %d", contentLength, uint16(MaxContentSize))
	} else {
		binary.LittleEndian.PutUint16(buf[tagBase+tagsSectionLength:], uint16(contentLength))
	}
	copy(buf[tagBase+tagsSectionLength+2:], []byte(evt.Content))

	return nil
}

func Unmarshal(data []byte, evt *nostr.Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to decode binary: %v", r)
		}
	}()

	evt.Kind = nostr.Kind(binary.LittleEndian.Uint16(data[1:3]))
	evt.CreatedAt = nostr.Timestamp(binary.LittleEndian.Uint32(data[3:7]))
	evt.ID = nostr.ID(data[7:39])
	evt.PubKey = nostr.PubKey(data[39:71])
	evt.Sig = [64]byte(data[71:135])

	const tagbase = 135
	tagsSectionLength := int(binary.LittleEndian.Uint16(data[tagbase:]))
	ntags := binary.LittleEndian.Uint16(data[tagbase+2:])
	evt.Tags = make(nostr.Tags, ntags)
	prevOffset := 0
	overflows := 0
	for t := range evt.Tags {
		offset := int(binary.LittleEndian.Uint16(data[tagbase+4+t*2:]))
		if offset < prevOffset {
			// we've reached the u16 overflow for the tag offsets, so we do this amazing hack
			overflows++
		}
		prevOffset = offset
		offset += (1 << 16) * overflows

		nitems := int(data[tagbase+offset])
		tag := make(nostr.Tag, nitems)

		curr := tagbase + offset + 1
		for i := range tag {
			length := int(binary.LittleEndian.Uint16(data[curr:]))
			tag[i] = string(data[curr+2 : curr+2+length])
			curr += 2 + length
		}
		evt.Tags[t] = tag
	}

	tagsSectionLength += (1 << 16) * overflows
	contentLength := int(binary.LittleEndian.Uint16(data[tagbase+tagsSectionLength:]))
	evt.Content = string(data[tagbase+tagsSectionLength+2 : tagbase+tagsSectionLength+2+contentLength])

	return err
}

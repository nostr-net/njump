package data

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/exp/slices"
)

// ParseRangeTable parses the specified Unicode.org data file for characters in the
// specified group, and returns a range table of those characters.
//
// Note that the range table reflects the ranges as defined in the source files; ranges
// are guaranteed not to overlap, as per the RangeTable docs, but adjacent ranges are not
// coalesced.
func ParseRangeTable(group EmojiGroup) *unicode.RangeTable {
	var re *regexp.Regexp
	switch group {
	case BasicEmoji:
		re = BasicEmojiGroupRegex
	case EmojiKeyCapSequence:
		re = EmojiKeyCapSequenceGroupRegex
	case EmojiFlagSequence:
		re = EmojiFlagSequenceGroupRegex
	case EmojiTagSequence:
		re = EmojiTagSequenceGroupRegex
	case EmojiModifierSequence:
		re = EmojiModifierSequenceGroupRegex
	case EmojiZWJSequence:
		re = EmojiZWJSequenceGroupRegex
	default:
		panic("exhaustive switch")
	}

	var r16s []unicode.Range16
	var r32s []unicode.Range32

	for _, line := range strings.Split(Data, "\n") {
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		if !re.MatchString(line) {
			continue
		}
		start, end, ok := toRange(line)
		if !ok {
			continue
		}
		r16, err := parseRange16(start, end)
		if err == nil {
			r16s = append(r16s, *r16)
			continue
		}
		r32, err := parseRange32(start, end)
		if err == nil {
			r32s = append(r32s, *r32)
		}
	}

	slices.SortFunc(r16s, func(a, b unicode.Range16) int { return int(a.Lo) - int(b.Lo) })
	slices.SortFunc(r32s, func(a, b unicode.Range32) int { return int(a.Lo) - int(b.Lo) })

	latinOffset := 0
	for _, r16 := range r16s {
		if r16.Hi <= unicode.MaxLatin1 {
			latinOffset++
		}
	}

	return &unicode.RangeTable{
		R16:         r16s,
		R32:         r32s,
		LatinOffset: latinOffset,
	}
}

func toRange(line string) (start, end string, ok bool) {
	rangeMatch := rangeRegexp.FindStringSubmatch(line)
	if len(rangeMatch) > 1 {
		start = rangeMatch[1]
		end = rangeMatch[2]
		return start, end, true
	} else {
		if singleMatch := singleRegexp.FindStringSubmatch(line); len(singleMatch) > 1 {
			start = singleMatch[1]
			end = singleMatch[1]
			return start, end, true
		}
	}
	return "", "", false
}

func parseRange16(startStr, endStr string) (*unicode.Range16, error) {
	start, err := parse16(startStr)
	if err != nil {
		return nil, err
	}
	end, err := parse16(endStr)
	if err != nil {
		return nil, err
	}
	r16 := unicode.Range16{
		Lo:     *start,
		Hi:     *end,
		Stride: 1,
	}
	return &r16, nil
}

func parse16(str string) (*uint16, error) {
	val, err := strconv.ParseInt(str, 16, 16)
	if err != nil {
		return nil, err
	}
	if val < math.MaxUint16 {
		val16 := uint16(val)
		return &val16, nil
	}
	return nil, fmt.Errorf("value %#v (%X) cannot be parsed as uint16", str, val)
}

func parseRange32(startStr, endStr string) (*unicode.Range32, error) {
	start, err := parse32(startStr)
	if err != nil {
		return nil, err
	}
	end, err := parse32(endStr)
	if err != nil {
		return nil, err
	}
	r32 := unicode.Range32{
		Lo:     *start,
		Hi:     *end,
		Stride: 1,
	}
	return &r32, nil
}

func parse32(str string) (*uint32, error) {
	val, err := strconv.ParseInt(str, 16, 32)
	if err != nil {
		return nil, err
	}
	if val < math.MaxUint32 {
		val32 := uint32(val)
		return &val32, nil
	}
	return nil, fmt.Errorf("value %#v (%X) cannot be parsed as uint32", str, val)
}

package emoji

import (
	"unicode"

	"github.com/nbd-wtf/emoji/data"
)

var emoji = data.ParseRangeTable(data.BasicEmoji)

// this is the range of base emojis, it also includes things that are used only in combinations,
// like "the red hair"
func IsEmoji(r rune) bool { return unicode.Is(emoji, r) }

// skin tone modifiers are applied right after emojis to change their skin tones
func IsSkinToneModifier(r rune) bool { return unicode.Is(data.EmojiSkinToneModifier, r) }

// the ZWJ is used to combine emojis into a single image, like "couple + man + woman"
func IsZeroWidthJoiner(r rune) bool { return r == '\u200d' }

// the variation selector is a useless thing that tells the renderer if it should display an
// image or a text format of the emoji, can be safely ignored
func IsVariationSelector(r rune) bool { return r == '\uFE0F' || r == '\uFE0E' }

// regional indicators are used to make flags: each regional indicator represents a letter,
// if you combine one that represents "B" with one that represents "R" you get the flag of
// Brazil
func IsRegionalIndicator(r rune) bool { return unicode.Is(data.RegionalIndicator, r) }

// tags are used for subnational flags like the flag of Wales, they consist of the black flag emoji
// followed by 5 tag runes that work like the country flags, but specifying the ISO-something
// code for subnational regions (Wales is "gbwls")
func IsTag(r rune) bool { return unicode.Is(data.Tag, r) }

// GetNextEmojiCharacters searches for an emoji sequence in the beginning of the given runes slice and returns
// the number of characters that single emoji is (or should be if it's implemented according to the
// crazy spec) comprised of
func GetNextEmojiCharacters(runes []rune, n int) int {
	if n == 0 {
		return 0
	}

	// if it starts with a normal emoji it can be multiple things
	if IsEmoji(runes[0]) {
		if n == 1 {
			// if there are no more runes we stop here anyway
			return 1
		}

		i := 1

		// if it is followed by a tag it indicates this is a subnational flag
		if IsTag(runes[i]) {
			// it can be followed by any number of tags
			is := n - i
			for ; i < is; i++ {
				if !IsTag(runes[i]) {
					i-- // the last read run should not be counted since it isn't a tag
					break
				}
			}
			return i
		}

		// it can be followed by a single skin tone modifier
		if i < n && IsSkinToneModifier(runes[i]) {
			i++
		}

		// it can also be followed by a display variation selector
		if i < n && IsVariationSelector(runes[i]) {
			i++
		}

		// if it is followed by a ZWJ that means we will see another emoji after that
		if i < n && IsZeroWidthJoiner(runes[i]) {
			i++
			return i + GetNextEmojiCharacters(runes[i:], n-i)
		}

		return i
	}

	// if it starts with a flag character it can only be a flag with 1 (buggt) or 2 characters (normal)
	if IsRegionalIndicator(runes[0]) {
		i := 1
		if n >= 2 && IsRegionalIndicator(runes[1]) {
			i++
		}
		return i
	}

	return 0
}

// GetNextEmoji searches for an emoji sequence in the beginning of the given runes slice and returns the
// runes corresponding to that
func GetNextEmoji(runes []rune) []rune {
	chars := GetNextEmojiCharacters(runes, len(runes))
	return runes[0:chars]
}

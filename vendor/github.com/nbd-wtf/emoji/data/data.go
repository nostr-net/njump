package data

import (
	_ "embed"
	"fmt"
	"regexp"
)

//go:embed emoji-sequences.txt
var EmojiSequences string

//go:embed emoji-zwj-sequences.txt
var EmojiZwjSequences string

var Data = EmojiSequences + "\n" + EmojiZwjSequences

const (
	cp = "[A-F0-9]{4,5}"
)

var (
	rangeRegexp  = regexp.MustCompile("^(" + cp + ")[.]{2}(" + cp + ")")
	singleRegexp = regexp.MustCompile("^(" + cp + ")")
	seqRegexp    = regexp.MustCompile("^" + cp + "(?: " + cp + ")+")
)

type EmojiGroup string

const (
	BasicEmoji            EmojiGroup = "Basic_Emoji"
	EmojiKeyCapSequence   EmojiGroup = "Emoji_Keycap_Sequence"
	EmojiFlagSequence     EmojiGroup = "RGI_Emoji_Flag_Sequence"
	EmojiTagSequence      EmojiGroup = "RGI_Emoji_Tag_Sequence;"
	EmojiModifierSequence EmojiGroup = "RGI_Emoji_Modifier_Sequence"
	EmojiZWJSequence      EmojiGroup = "RGI_Emoji_ZWJ_Sequence"
)

var (
	BasicEmojiGroupRegex            = regexp.MustCompile(fmt.Sprintf(";\\s+%v\\s*[;#]", BasicEmoji))
	EmojiKeyCapSequenceGroupRegex   = regexp.MustCompile(fmt.Sprintf(";\\s+%v\\s*[;#]", EmojiKeyCapSequence))
	EmojiFlagSequenceGroupRegex     = regexp.MustCompile(fmt.Sprintf(";\\s+%v\\s*[;#]", EmojiFlagSequence))
	EmojiTagSequenceGroupRegex      = regexp.MustCompile(fmt.Sprintf(";\\s+%v\\s*[;#]", EmojiTagSequence))
	EmojiModifierSequenceGroupRegex = regexp.MustCompile(fmt.Sprintf(";\\s+%v\\s*[;#]", EmojiModifierSequence))
	EmojiZWJSequenceGroupRegex      = regexp.MustCompile(fmt.Sprintf(";\\s+%v\\s*[;#]", EmojiZWJSequence))
)

var (
	BasicEmojiGroup            = parseSequencesMatching(BasicEmojiGroupRegex)
	EmojiKeyCapSequenceGroup   = parseSequencesMatching(EmojiKeyCapSequenceGroupRegex)
	EmojiFlagSequenceGroup     = parseSequencesMatching(EmojiFlagSequenceGroupRegex)
	EmojiTagSequenceGroup      = parseSequencesMatching(EmojiTagSequenceGroupRegex)
	EmojiModifierSequenceGroup = parseSequencesMatching(EmojiModifierSequenceGroupRegex)
	EmojiZWJSequenceGroup      = parseSequencesMatching(EmojiZWJSequenceGroupRegex)
)

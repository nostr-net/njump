package data

import (
	"regexp"
	"strconv"
	"strings"
)

func parseSequencesMatching(re *regexp.Regexp) []string {
	var result []string
	for _, line := range strings.Split(Data, "\n") {
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		if !re.MatchString(line) {
			continue
		}
		seq, ok := toSeq(line)
		if !ok {
			continue
		}
		result = append(result, seq)
	}
	return result
}

func toSeq(line string) (string, bool) {
	seqMatch := seqRegexp.FindStringSubmatch(line)
	if len(seqMatch) == 1 {
		seq, err := parseSeq(strings.Split(seqMatch[0], " "))
		if err != nil {
			return "", false
		}
		return seq, true
	}
	return "", false
}

func parseSeq(seq []string) (string, error) {
	var result []rune
	for _, s := range seq {
		val, err := strconv.ParseInt(s, 16, 64)
		if err != nil {
			return "", err
		}
		result = append(result, rune(val))
	}
	return string(result), nil
}

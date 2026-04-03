package main

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"fiatjaf.com/nostr"
	"fiatjaf.com/nostr/sdk"
)

func isMaliciousBridged(pm sdk.ProfileMetadata) bool {
	return strings.Contains(pm.NIP05, "rape.pet") || strings.Contains(pm.NIP05, "rape-pet")
}

func hasProhibitedWordOrTag(event *nostr.Event) bool {
	for _, tag := range event.Tags {
		if len(tag) >= 2 && tag[0] == "t" && slices.Contains(pornTags, strings.ToLower(tag[1])) {
			return true
		}
	}

	return pornWordsRe.MatchString(event.Content)
}

// hasExplicitMedia checks if the event contains explicit media content
// by examining image/video URLs in the content and checking them against the media alert API.
// Checks run in parallel with a bounded total timeout to avoid blocking the request.
func hasExplicitMedia(ctx context.Context, event *nostr.Event) bool {
	// extract image and video URLs from content
	var mediaURLs []string

	// find image URLs
	imgMatches := imageExtensionMatcher.FindAllStringSubmatch(event.Content, -1)
	for _, match := range imgMatches {
		if len(match) > 0 {
			mediaURLs = append(mediaURLs, match[0])
		}
	}

	// find video URLs
	vidMatches := videoExtensionMatcher.FindAllStringSubmatch(event.Content, -1)
	for _, match := range vidMatches {
		if len(match) > 0 {
			mediaURLs = append(mediaURLs, match[0])
		}
	}

	if len(mediaURLs) == 0 {
		return false
	}

	// cap total media checking time at 3s to avoid eating the request timeout
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// check all URLs in parallel
	type result struct {
		explicit bool
		err      error
		url      string
	}
	results := make(chan result, len(mediaURLs))

	for _, mediaURL := range mediaURLs {
		go func(u string) {
			isExplicit, err := isExplicitContent(checkCtx, u)
			results <- result{explicit: isExplicit, err: err, url: u}
		}(mediaURL)
	}

	for i := 0; i < len(mediaURLs); i++ {
		select {
		case r := <-results:
			if r.err != nil {
				log.Warn().Err(r.err).Str("url", r.url).Msg("failed to check media content")
				continue
			}
			if r.explicit {
				return true
			}
		case <-checkCtx.Done():
			return false
		}
	}

	return false
}

// list copied from https://jsr.io/@gleasonator/policy/0.9.8/policies/AntiPornPolicy.ts
var pornTags = []string{
	"adult",
	"ass",
	"assworship",
	"boobs",
	"boobies",
	"butt",
	"cock",
	"dick",
	"dickpic",
	"explosionloli",
	"femboi",
	"femboy",
	"fetish",
	"fuck",
	"freeporn",
	"girls",
	"loli",
	"milf",
	"nude",
	"nudity",
	"nsfw",
	"pantsu",
	"pussy",
	"porn",
	"porno",
	"porntube",
	"pornvideo",
	"sex",
	"sexpervertsyndicate",
	"sexporn",
	"sexy",
	"slut",
	"teen",
	"tits",
	"teenporn",
	"teens",
	"transnsfw",
	"xxx",
	"うちの子を置くとみんながうちの子に対する印象をリアクションしてくれるタグ",
}

var pornWordsRe = func() *regexp.Regexp {
	// list copied from https://jsr.io/@gleasonator/policy/0.2.0/data/pornwords.json
	pornWords := []string{
		"loli",
		"nsfw",
		"teen porn",
	}
	concat := strings.Join(pornWords, "|")
	regex := fmt.Sprintf(`\b(%s)\b`, concat)
	return regexp.MustCompile(regex)
}()

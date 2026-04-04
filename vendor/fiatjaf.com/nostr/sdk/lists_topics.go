package sdk

import (
	"context"

	"fiatjaf.com/nostr"
	cache_memory "fiatjaf.com/nostr/sdk/cache/memory"
)

type Topic string

func (r Topic) Value() string { return string(r) }

func (sys *System) FetchTopicList(ctx context.Context, pubkey nostr.PubKey) GenericList[string, Topic] {
	sys.topicListCacheOnce.Do(func() {
		if sys.TopicListCache == nil {
			sys.TopicListCache = cache_memory.New[GenericList[string, Topic]](1000)
		}
	})

	ml, _ := fetchGenericList(sys, ctx, pubkey, 10015, kind_10015, parseTopicString, sys.TopicListCache)
	return ml
}

func (sys *System) FetchTopicSets(ctx context.Context, pubkey nostr.PubKey) GenericSets[string, Topic] {
	sys.topicSetsCacheOnce.Do(func() {
		if sys.TopicSetsCache == nil {
			sys.TopicSetsCache = cache_memory.New[GenericSets[string, Topic]](1000)
		}
	})

	ml, _ := fetchGenericSets(sys, ctx, pubkey, 30015, kind_30015, parseTopicString, sys.TopicSetsCache)
	return ml
}

func parseTopicString(tag nostr.Tag) (t Topic, ok bool) {
	if len(tag) < 2 {
		return t, false
	}
	if t := tag[1]; t != "" && tag[0] == "t" {
		return Topic(t), true
	}

	return t, false
}

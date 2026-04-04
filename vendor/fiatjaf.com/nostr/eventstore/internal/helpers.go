package internal

import (
	"math"

	"fiatjaf.com/nostr"
)

func ChooseNarrowestTag(filter nostr.Filter) (key string, values []string, goodness int) {
	var tagKey string
	var tagValues []string
	for key, values := range filter.Tags {
		switch key {
		case "e", "E", "q":
			// 'e' and 'q' are the narrowest possible, so if we have that we will use it and that's it
			tagKey = key
			tagValues = values
			goodness = 9
			break
		case "a", "A", "i", "I", "g", "r":
			// these are second-best as they refer to relatively static things
			goodness = 8
			tagKey = key
			tagValues = values
		case "d":
			// this is as good as long as we have an "authors"
			if len(filter.Authors) != 0 && goodness < 7 {
				goodness = 7
				tagKey = key
				tagValues = values
			} else if goodness < 4 {
				goodness = 4
				tagKey = key
				tagValues = values
			}
		case "h", "t", "l", "k", "K":
			// these things denote "categories", so they are a little more broad
			if goodness < 6 {
				goodness = 6
				tagKey = key
				tagValues = values
			}
		case "p":
			// this is broad and useless for a pure tag search, but we will still prefer it over others
			// for secondary filtering
			if goodness < 2 {
				goodness = 2
				tagKey = key
				tagValues = values
			}
		default:
			// all the other tags are probably too broad and useless
			if goodness == 0 {
				tagKey = key
				tagValues = values
			}
		}
	}

	return tagKey, tagValues, goodness
}

func CopyMapWithoutKey[K comparable, V any](originalMap map[K]V, key K) map[K]V {
	newMap := make(map[K]V, len(originalMap)-1)
	for k, v := range originalMap {
		if k != key {
			newMap[k] = v
		}
	}
	return newMap
}

// BatchSizePerNumberOfQueries tries to make an educated guess for the batch size given the total filter limit and
// the number of abstract queries we'll be conducting at the same time
func BatchSizePerNumberOfQueries(totalFilterLimit int, numberOfQueries int) int {
	if numberOfQueries == 1 || totalFilterLimit*numberOfQueries < 50 {
		return totalFilterLimit
	}

	return max(
		4,
		int(
			math.Ceil(
				math.Pow(float64(totalFilterLimit), 0.80)/math.Pow(float64(numberOfQueries), 0.71),
			),
		),
	)
}

func SwapDelete[A any](arr []A, i int) []A {
	arr[i] = arr[len(arr)-1]
	return arr[:len(arr)-1]
}

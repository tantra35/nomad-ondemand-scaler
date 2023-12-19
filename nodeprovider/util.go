package nodeprovider

import (
	"math/rand"
	"regexp"
	"time"

	nomad "github.com/hashicorp/nomad/api"
)

func Min[T int | int32 | float32](a T, b T) T {
	if a < b {
		return a
	}
	return b
}

func CompareSlices[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func RemoveSliceElements[T comparable](slice []T, elements []T) []T {
	result := make([]T, 0, len(slice))

	for _, item := range slice {
		found := false
		for _, element := range elements {
			if item == element {
				found = true
				break
			}
		}

		if !found {
			result = append(result, item)
		}
	}

	return result
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateRandomString(length int) string {
	b := make([]rune, length)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func GetLastRegisterEvent(_events []*nomad.NodeEvent) *nomad.NodeEvent {
	var llastregisterevnt *nomad.NodeEvent
	regregexp := regexp.MustCompile(`Node.*registered`)

	for _, levnt := range _events {
		if regregexp.MatchString(levnt.Message) {
			llastregisterevnt = levnt
		}
	}

	if llastregisterevnt == nil {
		llastregisterevnt = _events[0]
	}

	return llastregisterevnt
}

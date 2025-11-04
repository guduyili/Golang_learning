package consistenthash

import (
	"strconv"
	"testing"
)

func TestHashing(t *testing.T) {
	hash := New(3, func(key []byte) uint32 {
		i, _ := strconv.Atoi(string(key))
		return uint32(i)
	})

	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	//02 12 22 04 14 24 06 16 26
	// 2 4 6 12 14 16 22 24 26
	hash.Add("6", "4", "2")

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

	//08 18 28
	hash.Add("8")

	// 由于虚拟节点的插入，27 现在应该映射到新节点 "8"
	// testCases["27"] = "8"
	testCases["27"] = "4"

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}
}

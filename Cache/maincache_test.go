// 负责与外部交互，控制缓存存储和获取的主流程

package cache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Fatalf("Getter failed")
	}
}

var db = map[string]string{
	"jw":    "114",
	"boyue": "514",
	"Sam":   "567",
}

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))
	tmp := NewGroup("test", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range db {
		if view, err := tmp.Get(k); err != nil || view.String() != v {
			t.Fatalf("failed to get value of %s", k)
		}

		if _, err := tmp.Get(k); err != nil || loadCounts[k] != 1 {
			t.Fatalf("cache %s miss", k)
		}
	}

	if view, err := tmp.Get("unknown"); err == nil {
		t.Fatalf("the value of unknown should be nil, but %s got", view)
	}
}

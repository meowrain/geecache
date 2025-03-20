package singleflight

import (
	"fmt"
	"sync"
	"testing"
)

func TestDo(t *testing.T) {
	var g Group
	var count int = 0
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		v, err := g.Do("Tom", func() (any, error) {
			count++
			return "bar", nil
		})
		if v != "bar" || err != nil {
			t.Errorf("Do v = %v,error = %v", v, err)
		}
		wg.Done()
	}()
	go func() {
		v, err := g.Do("Tom", func() (any, error) {
			count++
			return "bar", nil
		})
		if v != "bar" || err != nil {
			t.Errorf("Do v = %v,error = %v", v, err)
		}
		wg.Done()
	}()
	wg.Wait()
	fmt.Println(count)
	if count != 1 {
		t.Errorf("count = %v, want 1", count)
	}
}

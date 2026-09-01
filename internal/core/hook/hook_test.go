package hook

import (
	"math"
	"strings"
	"sync"
	"testing"
)

func TestStringHook_Apply(t *testing.T) {
	hook := NewHook[string]()

	hook.Add(func(s string) string {
		return "[Go] " + s
	})

	hook.Add(func(s string) string {
		return strings.ToLower(s)
	})

	result := hook.Apply("Hook System")
	expected := "[go] hook system"

	if result != expected {
		t.Errorf("Expected: %q, Got: %q", expected, result)
	}
}

func TestStructHook_Apply(t *testing.T) {
	type Product struct {
		Name  string
		Price float64
	}

	hook := NewHook[Product]()

	hook.Add(func(p Product) Product {
		p.Price = p.Price * 1.10
		return p
	})

	hook.Add(func(p Product) Product {
		p.Name = strings.ToUpper(p.Name)
		return p
	})

	input := Product{Name: "Book", Price: 100.0}
	result := hook.Apply(input)

	diff := math.Abs(result.Price - 110.0)
	if diff > 0.00001 {
		t.Errorf("Expected Price: 110.0, Got: %f", result.Price)
	}

	if result.Name != "BOOK" {
		t.Errorf("Expected Name: 'BOOK', Got: %q", result.Name)
	}
}

func TestHook_ConcurrentSafety(t *testing.T) {
	hook := NewHook[int]()
	var wg sync.WaitGroup

	hook.Add(func(n int) int {
		return n + 1
	})

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hook.Add(func(n int) int {
				return n + 0
			})
		}()
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := hook.Apply(10)
			if res < 11 {
				t.Errorf("Concurrency error: Result %d is less than expected base value", res)
			}
		}()
	}

	wg.Wait()
}

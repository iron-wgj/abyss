package collector

import (
	"fmt"
	"testing"
)

func TestCounterAdd(t *testing.T) {
	counter := NewCounter(CounterOpts{
		Name:        "test",
		Help:        "test help",
		ConstLabels: Labels{"a": "1", "b": "2"},
		Level:       LevelLog,
	}).(*counter)
	expectedValue := counter.value
	counter.Inc()
	expectedValue++
	if expected, got := uint64(expectedValue), counter.value; expected != got {
		t.Errorf("Expected %d, got %d.", expected, got)
	}

	counter.Add(42)
	expectedValue += 42
	if expected, got := uint64(expectedValue), counter.value; expected != got {
		t.Errorf("Expected %d, got %d.", expected, got)
	}

	//if err := decreaseCounter(counter); err == nil {
	//	t.Errorf("Counter must panic when add a negative integer.")
	//}

	m, err := counter.Write()
	if err != nil {
		t.Errorf(err.Error())
	}

	if expected, got := fmt.Sprintf("label:{name:\"a\" value:\"1\"} label:{name:\"b\" value:\"2\"} counter:{value:%d}", expectedValue), m.String(); expected != got {
		t.Log(got)
		t.Errorf("Expected %q, got %q.", expected, got)
	}
}

func TestCounterParell(t *testing.T) {
	counter := NewCounter(CounterOpts{
		Name:        "test",
		Help:        "test help",
		ConstLabels: Labels{"a": "1", "b": "2"},
		Level:       LevelLog,
	}).(*counter)
	expectedValue := counter.value
	t.Parallel()
	t.Run("Inc", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			counter.Inc()
		}
	})
	t.Run("Add", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			counter.Add(float64(i))
		}
	})

	expectedValue += 10
	for i := 0; i < 5; i++ {
		expectedValue += uint64(i)
	}
	if expectedValue != counter.value {
		t.Errorf("Expected %d, get %d.", expectedValue, counter.value)
	}
}

func decreaseCounter(c *counter) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()
	c.Add(-1)
	return nil
}

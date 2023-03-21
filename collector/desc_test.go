package collector

import (
	"testing"
)

func TestNewDesc(t *testing.T) {
	desc := NewDesc(
		"ssss",
		"ssss",
		LevelInfo,
		Labels{"a": "aaaa"},
		Labels{"b": "bbbb"},
	)
	t.Log(desc.String())
}

func TestNewDescDupLabelName(t *testing.T) {
	desc := NewDesc(
		"ssss",
		"ssss",
		LevelInfo,
		Labels{"a": "aaaa"},
		Labels{"a": "bbbb"},
	)
	if desc.err == nil {
		t.Errorf("Expect Desc should be error, result is %s.", desc.String())
	} else {
		t.Logf("Get duplicated error: %s.", desc.err.Error())
	}
}

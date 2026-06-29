package utils

import (
	"bytes"
	"testing"
)

func TestDd(t *testing.T) {
	buf := bytes.NewBuffer([]byte("abcdefghh"))
	err := Dd(buf, "/tmp/test-dd", 4, int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
}

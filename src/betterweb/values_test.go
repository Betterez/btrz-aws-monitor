package betterweb

import (
	"testing"
	"time"
)

func TestTimeValues(t *testing.T) {
	t1 := time.Now()
	t2 := time.Now()
	t1.Add(time.Minute * 1)
	t3 := t1.Sub(t2)
	if t3.Minutes() != 1 {
		t.Errorf("time difference failure: should be 1, but it is %f", t3.Minutes())
	}
}

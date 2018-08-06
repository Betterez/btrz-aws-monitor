package betterweb

import (
	"testing"
	"time"
)

func TestTimeValues(t *testing.T) {
	t1 := time.Now()
	t2 := t1.Add(time.Minute * 1)
	t3 := t2.Sub(t1)
	if t3.Minutes() != 1 {
		t.Errorf("time difference failure: should be 1, but it is %f", t3.Minutes())
	}
}

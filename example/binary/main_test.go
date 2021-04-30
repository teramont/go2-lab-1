package binary

import "testing"

func TestMain(t *testing.T) {
	res := HelloWorld()

	if res != 42 {
		t.Errorf("Result is incorrect")
	}
}

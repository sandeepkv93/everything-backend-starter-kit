package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"
)

func TestPrintCIResultJSONOutput(t *testing.T) {
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	PrintCIResult(false, "seed apply", []string{"x", "y"}, errors.New("boom"))
	_ = w.Close()
	os.Stdout = orig

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		t.Fatalf("read output: %v", err)
	}
	_ = r.Close()

	var got CIResult
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v; raw=%q", err, buf.String())
	}
	if got.OK || got.Title != "seed apply" || got.Error != "boom" || len(got.Details) != 2 {
		t.Fatalf("unexpected ci result: %+v", got)
	}
}

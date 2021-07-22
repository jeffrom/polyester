package compiler

import (
	"context"
	"testing"

	"github.com/jeffrom/polyester/manifest"
	"github.com/jeffrom/polyester/testenv"
)

func TestCompiler(t *testing.T) {
	cc := New()
	m, err := manifest.LoadDir(testenv.Path("testdata", "basic"))
	if err != nil {
		t.Fatal("load manifest failed:", err)
	}
	if m == nil {
		t.Fatal("manifest was nil")
	}

	plan, err := cc.Compile(context.Background(), m)
	if err != nil {
		t.Fatal("compile failed:", err)
	}
	if plan == nil {
		t.Fatal("plan was nil")
	}

	sorted, err := plan.All()
	if err != nil {
		t.Fatal("dep resolve failed:", err)
	}

	firstTouchIdx := -1
	touchIdx := -1
	for i, pl := range sorted {
		// fmt.Println(i, pl)
		if pl.Name == "first-touch" {
			firstTouchIdx = i
		} else if pl.Name == "touchy" {
			touchIdx = i
		}
	}

	if firstTouchIdx == -1 {
		t.Fatal("didn't find first-touch in sorted plans")
	}
	if touchIdx == -1 {
		t.Fatal("didn't find touchy in sorted plans")
	}
	if firstTouchIdx > touchIdx {
		t.Errorf("expected first-touch before touchy. first-touch was #%d, touchy was #%d", firstTouchIdx, touchIdx)
	}
}

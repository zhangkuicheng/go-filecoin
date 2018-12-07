package process

import (
	"context"
	"io/ioutil"
	"testing"
)

func TestSomething(t *testing.T) {
	ctx := context.Background()
	dir, err := ioutil.TempDir("", "ACTION")
	if err != nil {
		panic(err)
	}
	env, err := NewEnvironment(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := env.AddGenesisMiner(ctx); err != nil {
		t.Fatal(err)
	}

	if err := env.AddNode(ctx); err != nil {
		t.Fatal(err)
	}

	if err := env.AddMiner(ctx); err != nil {
		t.Fatal(err)
	}
}

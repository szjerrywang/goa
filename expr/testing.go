package expr

import (
	"testing"

	"goa.design/goa/eval"
)

// RunHTTPDSL returns the http DSL root resulting from running the given DSL.
func RunHTTPDSL(t *testing.T, dsl func()) *RootExpr {
	setupHTTPDSLRun()

	// run DSL (first pass)
	if !eval.Execute(dsl, nil) {
		t.Fatal(eval.Context.Error())
	}

	// run DSL (second pass)
	if err := eval.RunDSL(); err != nil {
		t.Fatal(err)
	}

	// return generated root
	return Root
}

// RunInvalidHTTPDSL returns the error resulting from running the given DSL.
func RunInvalidHTTPDSL(t *testing.T, dsl func()) error {
	setupHTTPDSLRun()

	// run DSL (first pass)
	if !eval.Execute(dsl, nil) {
		return eval.Context.Errors
	}

	// run DSL (second pass)
	if err := eval.RunDSL(); err != nil {
		return err
	}

	// expected an error - didn't get one
	t.Fatal("expected a DSL evaluation error - got none")

	return nil
}

func setupHTTPDSLRun() {
	// reset all roots and codegen data structures
	eval.Reset()
	Root = new(RootExpr)
	Root.GeneratedTypes = &GeneratedRoot{}
	eval.Register(Root)
	eval.Register(Root.GeneratedTypes)
	eval.Register(Root)
	Root.API = &APIExpr{
		Name:    "test api",
		Servers: []*ServerExpr{{URL: "http://localhost"}},
	}
}

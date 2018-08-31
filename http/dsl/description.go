package dsl

import (
	"goa.design/goa/dsl"
	"goa.design/goa/eval"
	"goa.design/goa/expr"
)

// Description sets the expression description.
//
// Description must appear in API, Service, Endpoint, Files, Response, Type,
// ResultType or Attribute.
//
// Description accepts a single argument which is the description value.
//
// Example:
//
//    var _ = API("cellar", func() {
//        Description("The wine cellar API")
//    })
//
func Description(d string) {
	switch e := eval.Current().(type) {
	case *expr.HTTPResponseExpr:
		e.Description = d
	case *expr.HTTPFileServerExpr:
		e.Description = d
	default:
		dsl.Description(d)
	}
}

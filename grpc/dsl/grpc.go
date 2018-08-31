package dsl

import (
	"goa.design/goa/eval"
	"goa.design/goa/expr"
	"google.golang.org/grpc/codes"
)

// GRPC defines gRPC transport specific properties on an API, a service, or a
// single method. The function maps the request and response types to gRPC
// properties such as request and response messages.
//
// As a special case GRPC may be used to define the response generated for
// invalid requests and internal errors (errors returned by the service methods
// that don't match any of the error responses defined in the design). This is
// the only use of GRPC allowed in the API expression.
//
// The functions that appear in GRPC such as Message or Response may take
// advantage of the request or response types (depending on whether they appear
// when describing the gRPC request or response). The properties of the message
// attributes inherit the properties of the attributes with the same names that
// appear in the request or response types. The functions may also define new
// attributes or override the existing request or response type attributes.
//
// GRPC must appear in API, a Service or a Method expression.
//
// GRPC accepts a single argument which is the defining DSL function.
//
// Example:
//
//    var _ = Service("calculator", func() {
//        Method("add", func() {
//            Description("Add two operands")
//            Payload(Operands)
//            Error(BadRequest, ErrorResult)
//
//            GRPC(func() {
//                Name("add")
//                Response(func() {
//                    Field(1, "sum", Integer, "The sum")
//                })
//            })
//        })
//    })
func GRPC(fn func()) {
	switch actual := eval.Current().(type) {
	case *expr.APIExpr:
		eval.Execute(fn, expr.Root)
	case *expr.ServiceExpr:
		res := expr.Root.GRPCServiceFor(actual)
		res.DSLFunc = fn
	case *expr.MethodExpr:
		res := expr.Root.GRPCServiceFor(actual.Service)
		act := res.EndpointFor(actual.Name, actual)
		act.DSLFunc = fn
	default:
		eval.IncompatibleDSL()
	}
}

// Message describes a gRPC request or response message.
//
// Message must appear in a Method gRPC expression to define the request
// message or in an Error or Result gRPC expression to define the response
// message. If Message is absent then the message is built using the gRPC
// endpoint request or response type attributes.
//
// Message accepts one argument which describes the shape of the body, it can be:
//
//  - The name of an attribute of the request or response type. In this case the
//    attribute type describes the shape of the message.
//
//  - A function listing the message attributes. The attributes inherit the
//    properties (description, type, validations etc.) of the request or
//    response type attributes with identical names.
//
// Assuming the type:
//
//     var CreatePayload = Type("CreatePayload", func() {
//         Attribute("name", String, "Name of account")
//     })
//
// The following:
//
//     Method("create", func() {
//         Payload(CreatePayload)
//				 GRPC(func() {
//         })
//     })
//
// is equivalent to:
//
//     Method("create", func() {
//         Payload(CreatePayload)
//         GRPC(func() {
//             Message(func() {
//                 Attribute("name")
//             })
//         })
//     })
//
func Message(args ...interface{}) {
	if len(args) == 0 {
		eval.ReportError("not enough arguments, use Message(name), Message(type), Message(func()) or Message(type, func())")
		return
	}

	var (
		ref       *expr.AttributeExpr
		setter    func(*expr.AttributeExpr)
		kind, tgt string
	)

	// Figure out reference type and setter function
	switch e := eval.Current().(type) {
	case *expr.GRPCEndpointExpr:
		ref = e.MethodExpr.Payload
		setter = func(att *expr.AttributeExpr) {
			e.Request = att
		}
		kind = "request"
		tgt = "Payload"
	case *expr.GRPCErrorExpr:
		ref = e.ErrorExpr.AttributeExpr
		setter = func(att *expr.AttributeExpr) {
			if e.Response == nil {
				e.Response = &expr.GRPCResponseExpr{}
			}
			e.Response.Message = att
		}
		kind = "error_" + e.Name
		tgt = "Error " + e.Name
	case *expr.GRPCResponseExpr:
		ref = e.Parent.(*expr.GRPCEndpointExpr).MethodExpr.Result
		setter = func(att *expr.AttributeExpr) {
			e.Message = att
		}
		kind = "response"
		tgt = "Result"
	default:
		eval.IncompatibleDSL()
		return
	}

	// Now initialize target attribute and DSL if any
	var (
		attr *expr.AttributeExpr
		fn   func()
	)
	switch a := args[0].(type) {
	case string:
		if ref.Find(a) == nil {
			eval.ReportError("%q is not found in %s", a, tgt)
			return
		}
		obj := expr.AsObject(ref.Type)
		if obj == nil {
			eval.ReportError("%s must be an object with an attribute with name %#v, got %T", tgt, a, ref.Type)
			return
		}
		attr = obj.Attribute(a)
		if attr == nil {
			eval.ReportError("%s does not have an attribute named %#v", tgt, a)
			return
		}
		attr = expr.DupAtt(attr)
		if attr.Metadata == nil {
			attr.Metadata = expr.MetadataExpr{"origin:attribute": []string{a}}
		} else {
			attr.Metadata["origin:attribute"] = []string{a}
		}
	case expr.UserType:
		attr = &expr.AttributeExpr{Type: a}
		if len(args) > 1 {
			var ok bool
			fn, ok = args[1].(func())
			if !ok {
				eval.ReportError("second argument must be a function")
			}
		}
	case func():
		fn = a
		if ref == nil {
			eval.ReportError("Message is set but %s is not defined", tgt)
			return
		}
		attr = ref
	default:
		eval.InvalidArgError("attribute name, user type or DSL", a)
		return
	}

	if fn != nil {
		eval.Execute(fn, attr)
	}
	if attr != nil {
		if attr.Metadata == nil {
			attr.Metadata = expr.MetadataExpr{}
		}
		attr.Metadata["grpc:"+kind] = []string{}
		setter(attr)
	}
}

// Response describes a gRPC response message. Response describes both success
// and error responses. When describing an error response the first argument is
// the name of the error. There can be at most one success response and any
// number of error responses.
//
// Response accepts one to three arguments. Success response accepts a status
// code as first argument. If the first argument is a status code then a
// function may be given as the second argument. This function can add more
// information to the response like adding description using Description,
// status code using Code, and the response message shape using Message.
//
// By default success gRPC response use code `OK` and error gRPC responses use
// code `Unknown`. Also by default the response uses the method result type
// (success response) or error type (error responses) to define the response
// message shape. See "google.golang.org/grpc/codes" package for gRPC status
// codes.
//
// Response must appear in an API or service GRPC expression to define error
// responses common to all the API or service methods. Response may also appear
// in a method GRPC expression to define both success and error responses
// specific to the method.
//
// The valid invocations for successful response are thus:
//
// * Response(status)
//
// * Response(func)
//
// * Response(status, func)
//
// Error responses additionally accept the name of the error as first argument.
//
// * Response(error_name, status)
//
// * Response(error_name, func)
//
// * Response(error_name, status, func)
//
// Example:
//
//    import "google.golang.org/grpc/codes"
//
//		// Response (success and error) message definition
//
//		Method("create", func() {
//				Payload(CreatePayload)
//				Result(CreateResult)
//				Error("an_error", String)
//
//				GRPC(func() {
//						Response(codes.OK)
//						Response("an_error", codes.Internal)
//				})
//		})
//
//		// Success response defined using func()
//
//    Method("create", func() {
//        Payload(CreatePayload)
//        Result(CreateResult)
//        Error("an_error")							// uses in-built ErrorResult type
//
//        GRPC(func() {
//            Response(func() {
//                Description("Response used when item already exists")
//                Code(codes.OK)				// gRPC status code set using Code
//                Message(CreateResult) // gRPC response set using Message
//            })
//
//						Response("an_error", codes.Unknown)	// error response
//        })
//    })
//
func Response(val interface{}, args ...interface{}) {
	name, ok := val.(string)
	switch t := eval.Current().(type) {
	case *expr.RootExpr:
		if !ok {
			eval.InvalidArgError("name of error", val)
			return
		}
		if e := grpcError(name, t, args...); e != nil {
			t.GRPCErrors = append(t.GRPCErrors, e)
		}
	case *expr.GRPCServiceExpr:
		if !ok {
			eval.InvalidArgError("name of error", val)
			return
		}
		if e := grpcError(name, t, args...); e != nil {
			t.GRPCErrors = append(t.GRPCErrors, e)
		}
	case *expr.GRPCEndpointExpr:
		if ok {
			// error response
			if e := grpcError(name, t, args...); e != nil {
				t.GRPCErrors = append(t.GRPCErrors, e)
			}
			return
		}
		code, fn := parseResponseArgs(val, args...)
		resp := &expr.GRPCResponseExpr{
			StatusCode: code,
			Parent:     t,
		}
		if fn != nil {
			eval.Execute(fn, resp)
		}
		t.Response = resp
	default:
		eval.IncompatibleDSL()
	}
}

// Code sets the Response status code. It must appear in a gRPC response
// expression.
func Code(code codes.Code) {
	res, ok := eval.Current().(*expr.GRPCResponseExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	res.StatusCode = code
}

func grpcError(n string, p eval.Expression, args ...interface{}) *expr.GRPCErrorExpr {
	if len(args) == 0 {
		eval.ReportError("not enough arguments, use Response(name, status), Response(name, status, func()) or Response(name, func())")
		return nil
	}
	var (
		code codes.Code
		fn   func()
		val  interface{}
	)
	val = args[0]
	args = args[1:]
	code, fn = parseResponseArgs(val, args...)
	if code == 0 {
		code = codes.Unknown
	}
	resp := &expr.GRPCResponseExpr{
		StatusCode: code,
		Parent:     p,
	}
	if fn != nil {
		eval.Execute(fn, resp)
	}
	return &expr.GRPCErrorExpr{
		Name:     n,
		Response: resp,
	}
}

func parseResponseArgs(val interface{}, args ...interface{}) (code codes.Code, fn func()) {
	switch t := val.(type) {
	case codes.Code:
		code = t
		if len(args) > 1 {
			eval.ReportError("too many arguments given to Response (%d)", len(args)+1)
			return
		}
		if len(args) == 1 {
			if d, ok := args[0].(func()); ok {
				fn = d
			} else {
				eval.InvalidArgError("function", args[0])
				return
			}
		}
	case func():
		if len(args) > 0 {
			eval.InvalidArgError("google.golang.org/grpc/codes.Code (gRPC status code)", val)
			return
		}
		fn = t
	default:
		eval.InvalidArgError("google.golang.org/grpc/codes.Code (gRPC status code) or function", val)
		return
	}
	return
}

package codegen

import (
	"fmt"

	"goa.design/goa/codegen"
	"goa.design/goa/codegen/service"
	"goa.design/goa/design"
	grpcdesign "goa.design/goa/grpc/design"
)

// GRPCServices holds the data computed from the design needed to generate the
// transport code of the services.
var GRPCServices = make(ServicesData)

type (
	// ServicesData encapsulates the data computed from the design.
	ServicesData map[string]*ServiceData

	// ServiceData contains the data used to render the code related to a
	// single service.
	ServiceData struct {
		// Service contains the related service data.
		Service *service.Data
		// PkgName is the name of the generated package in *.pb.go.
		PkgName string
		// Name is the service name.
		Name string
		// Description is the service description.
		Description string
		// Endpoints describes the gRPC service endpoints.
		Endpoints []*EndpointData
		// Messages describes the message data for this service.
		Messages []*MessageData
		// ServerStruct is the name of the gRPC server struct.
		ServerStruct string
		// ClientStruct is the name of the gRPC client struct,
		ClientStruct string
		// ServerInit is the name of the constructor of the server struct.
		ServerInit string
		// ClientInit is the name of the constructor of the client struct.
		ClientInit string
		// ServerInterface is the name of the gRPC server interface implemented
		// by the service.
		ServerInterface string
		// ClientInterface is the name of the gRPC client interface implemented
		// by the service.
		ClientInterface string
		// ClientInterfaceInit is the name of the client constructor function in
		// the generated pb.go package.
		ClientInterfaceInit string
		// TransformHelpers is the list of transform functions required by the
		// constructors.
		TransformHelpers []*codegen.TransformFunctionData
	}

	// EndpointData contains the data used to render the code related to
	// gRPC endpoint.
	EndpointData struct {
		// ServiceName is the name of the service.
		ServiceName string
		// PkgName is the name of the generated package in *.pb.go.
		PkgName string
		// Method is the data for the underlying method expression.
		Method *service.MethodData
		// PayloadRef is the fully qualified reference to the method payload.
		PayloadRef string
		// ResultRef is the fully qualified reference to the method result.
		ResultRef string
		// Request is the gRPC request data.
		Request *RequestData
		// Response is the gRPC response data.
		Response *ResponseData
		// Errors describes the method gRPC errors.
		Errors []*ErrorData

		// server side

		// ServerStruct is the name of the gRPC server struct.
		ServerStruct string
		// ServerInterface is the name of the gRPC server interface implemented
		// by the service.
		ServerInterface string

		// client side

		// ClientStruct is the name of the gRPC client struct,
		ClientStruct string
		// ClientInterface is the name of the gRPC client interface implemented
		// by the service.
		ClientInterface string
	}

	// MessageData contains the data used to render the code related to a
	// message for a gRPC service. It is used in generating the .proto file.
	// A gRPC request message is computed from the method Payload expression
	// (unless specified explicitly by Message DSL in the gRPC endpoint
	// expression). A gRPC response message is computed from the method Result
	// expression (unless specified explicitly by Message DSL in the gRPC
	// response expression). If the method Payload/Result is empty (not defined)
	// a corresponding message type with empty fields is still generated in the
	// .proto file. NOTE: Codegen could use `google.protobuf.Empty` message type
	// in such cases, but generating an empty message type for the request or
	// response makes it easier (and safer) if the request/response message needs
	// to be changed in the future.
	MessageData struct {
		// Name is the message name.
		Name string
		// Description is the message description.
		Description string
		// VarName is the variable name that holds the definition.
		VarName string
		// Def is the message definition.
		Def string
		// Type is the underlying type.
		Type design.UserType
	}

	// ErrorData contains the error information required to generate the
	// transport decode (client) and encode (server) code.
	ErrorData struct {
		// StatusCode is the response gRPC status code.
		StatusCode string
		// Name is the error name.
		Name string
		// Ref is a reference to the error type.
		Ref string
		// Response is the error response data.
		Response *ResponseData
	}

	// RequestData describes a gRPC request.
	RequestData struct {
		// Description is the request description.
		Description string
		// Message is the gRPC request message. It is used in generating
		// .proto file.
		Message *MessageData
		// ServerType is the request data with constructor function to
		// initialize the method payload type from the generated payload type in
		// *.pb.go.
		ServerType *TypeData
		// ClientType is the request data with constructor function to
		// initialize the generated payload type in *.pb.go from the
		// method payload.
		ClientType *TypeData
		// PayloadAttr sets the request message from the specified payload type
		// attribute. This field is set when the design uses Message("name") syntax
		// to set the request message and the payload type is an object.
		PayloadAttr string
	}

	// ResponseData describes a gRPC success or error response.
	ResponseData struct {
		// StatusCode is the return code of the response.
		StatusCode string
		// Description is the response description.
		Description string
		// Message is the gRPC response message. It is used in generating
		// .proto file.
		Message *MessageData
		// ServerType is the type data with constructor function to
		// initialize the generated response type in *.pb.go from the
		// method result type.
		ServerType *TypeData
		// ClientType is the type data with constructor function to
		// initialize the method result type from the generated response type in
		// *.pb.go.
		ClientType *TypeData
		// ResultAttr sets the response message from the specified result type
		// attribute. This field is set when the design uses Message("name") syntax
		// to set the response message and the result type is an object.
		ResultAttr string
	}

	// TypeData contains the request/response data and the constructor function
	// to initialize the type.
	// For request type, it contains data to transform gRPC request type to the
	// corresponding payload type (server) and vice versa (client).
	// For response type, it contains data to transform gRPC response type to the
	// corresponding result type (client) and vice versa (server).
	TypeData struct {
		// Name is the type name.
		Name string
		// Ref is the fully qualified reference to the type.
		Ref string
		// Init contains the data required to render the constructor if any.
		Init *InitData
	}

	// InitData contains the data required to render a constructor.
	InitData struct {
		// Name is the constructor function name.
		Name string
		// Description is the function description.
		Description string
		// Args is the list of constructor arguments.
		Args []*InitArgData
		// CLIArgs is the list of arguments for the command-line client.
		// This is set only for the client side.
		CLIArgs []*InitArgData
		// ReturnVarName is the name of the variable to be returned.
		ReturnVarName string
		// ReturnTypeRef is the qualified (including the package name)
		// reference to the return type.
		ReturnTypeRef string
		// Code is the transformation code.
		Code string
	}

	// InitArgData represents a single constructor argument.
	InitArgData struct {
		// Name is the argument name.
		Name string
		// Description is the argument description.
		Description string
		// Reference to the argument, e.g. "&body".
		Ref string
		// FieldName is the name of the data structure field that should
		// be initialized with the argument if any.
		FieldName string
		// TypeName is the argument type name.
		TypeName string
		// TypeRef is the argument type reference.
		TypeRef string
		// Pointer is true if a pointer to the arg should be used.
		Pointer bool
		// Required is true if the arg is required to build the payload.
		Required bool
		// DefaultValue is the default value of the arg.
		DefaultValue interface{}
		// Validate contains the validation code for the argument
		// value if any.
		Validate string
		// Example is a example value
		Example interface{}
	}
)

// Get retrieves the transport data for the service with the given name
// computing it if needed. It returns nil if there is no service with the given
// name.
func (d ServicesData) Get(name string) *ServiceData {
	if data, ok := d[name]; ok {
		return data
	}
	service := grpcdesign.Root.Service(name)
	if service == nil {
		return nil
	}
	d[name] = d.analyze(service)
	return d[name]
}

// Endpoint returns the service method transport data for the endpoint with the
// given name, nil if there isn't one.
func (sd *ServiceData) Endpoint(name string) *EndpointData {
	for _, ed := range sd.Endpoints {
		if ed.Method.Name == name {
			return ed
		}
	}
	return nil
}

// analyze creates the data necessary to render the code of the given service.
func (d ServicesData) analyze(gs *grpcdesign.ServiceExpr) *ServiceData {
	var (
		sd      *ServiceData
		seen    map[string]struct{}
		svcVarN string
		pkgName string

		svc = service.Services.Get(gs.Name())
	)
	{
		svcVarN = codegen.Goify(svc.Name, true)
		pkgName = svc.Name + "pb"
		sd = &ServiceData{
			Service:             svc,
			Name:                svc.Name,
			Description:         svc.Description,
			PkgName:             svc.Name + "pb",
			ServerStruct:        "Server",
			ClientStruct:        "Client",
			ServerInit:          "New",
			ClientInit:          "NewClient",
			ServerInterface:     svcVarN + "Server",
			ClientInterface:     svcVarN + "Client",
			ClientInterfaceInit: fmt.Sprintf("%s.New%sClient", pkgName, svcVarN),
		}
		seen = make(map[string]struct{})
	}
	for _, e := range gs.GRPCEndpoints {
		// Make request message to a user type
		if _, ok := e.Request.Type.(design.UserType); !ok {
			e.Request.Type = &design.UserTypeExpr{
				AttributeExpr: wrapAttr(e.Request),
				TypeName:      fmt.Sprintf("%sRequest", ProtoBufify(e.Name(), true)),
			}
		} else if e.Request.Type == design.Empty {
			// empty type should still generate a message. Rename the type to have
			// the endpoint name suffixed with Request.
			e.Request.Type = &design.UserTypeExpr{
				AttributeExpr: &design.AttributeExpr{Type: &design.Object{}},
				TypeName:      fmt.Sprintf("%sRequest", ProtoBufify(e.Name(), true)),
			}
		}

		// Make response message to a user type
		if _, ok := e.Response.Message.Type.(design.UserType); !ok {
			e.Response.Message.Type = &design.UserTypeExpr{
				AttributeExpr: wrapAttr(e.Response.Message),
				TypeName:      fmt.Sprintf("%sResponse", ProtoBufify(e.Name(), true)),
			}
		} else if e.Response.Message.Type == design.Empty {
			// empty type should still generate a message. Rename the type to have
			// the endpoint name suffixed with Response.
			e.Response.Message.Type = &design.UserTypeExpr{
				AttributeExpr: &design.AttributeExpr{Type: &design.Object{}},
				TypeName:      fmt.Sprintf("%sResponse", ProtoBufify(e.Name(), true)),
			}
		}

		// collect all the nested messages and return the top-level message
		collect := func(att *design.AttributeExpr) *MessageData {
			msgs := collectMessages(att, seen, svc.Scope)
			sd.Messages = append(sd.Messages, msgs...)
			return msgs[0]
		}

		var (
			request    *RequestData
			response   *ResponseData
			errors     []*ErrorData
			payloadRef string
			resultRef  string

			md = svc.Method(e.Name())
		)
		{
			request = &RequestData{
				Message:     collect(e.Request),
				Description: e.Request.Description,
				ServerType:  buildRequestTypeData(e, sd, true),
				ClientType:  buildRequestTypeData(e, sd, false),
			}
			if e.MethodExpr.Payload.Type != design.Empty {
				payloadRef = svc.Scope.GoFullTypeRef(e.MethodExpr.Payload, svc.PkgName)
			}
			response = &ResponseData{
				Message:     collect(e.Response.Message),
				StatusCode:  statusCodeToGRPCConst(e.Response.StatusCode),
				Description: e.Response.Description,
				ServerType:  buildResponseTypeData(e, sd, true),
				ClientType:  buildResponseTypeData(e, sd, false),
			}
			if e.MethodExpr.Result.Type != design.Empty {
				resultRef = svc.Scope.GoFullTypeRef(e.MethodExpr.Result, svc.PkgName)
			}
			errors = buildErrorsData(e, sd)
		}
		sd.Endpoints = append(sd.Endpoints, &EndpointData{
			ServiceName:     svc.Name,
			PkgName:         sd.PkgName,
			Method:          md,
			PayloadRef:      payloadRef,
			ResultRef:       resultRef,
			Request:         request,
			Response:        response,
			Errors:          errors,
			ServerStruct:    sd.ServerStruct,
			ServerInterface: sd.ServerInterface,
			ClientStruct:    sd.ClientStruct,
			ClientInterface: sd.ClientInterface,
		})
	}
	return sd
}

// wrapAttr wraps the given attribute into an attribute named "field" if
// the given attribute is a non-object type. For a raw object type it simply
// returns a dupped attribute.
func wrapAttr(att *design.AttributeExpr) *design.AttributeExpr {
	var attr *design.AttributeExpr
	switch actual := att.Type.(type) {
	case *design.Array:
	case *design.Map:
	case design.Primitive:
		attr = &design.AttributeExpr{
			Type: &design.Object{
				&design.NamedAttributeExpr{
					Name: "field",
					Attribute: &design.AttributeExpr{
						Type: actual,
						Meta: design.MetaExpr{"rpc:tag": []string{"1"}},
					},
				},
			},
		}
	case *design.Object:
		attr = design.DupAtt(att)
	}
	return attr
}

// collectMessages recurses through the attribute to gather all the messages.
func collectMessages(at *design.AttributeExpr, seen map[string]struct{}, scope *codegen.NameScope) (data []*MessageData) {
	if at == nil {
		return
	}
	collect := func(at *design.AttributeExpr) []*MessageData { return collectMessages(at, seen, scope) }
	switch dt := at.Type.(type) {
	case design.UserType:
		if _, ok := seen[dt.Name()]; ok {
			return nil
		}
		data = append(data, &MessageData{
			Name:        dt.Name(),
			VarName:     ProtoBufMessageName(at, scope),
			Description: dt.Attribute().Description,
			Def:         ProtoBufMessageDef(dt.Attribute(), scope),
			Type:        dt,
		})
		seen[dt.Name()] = struct{}{}
		data = append(data, collect(dt.Attribute())...)
	case *design.Object:
		for _, nat := range *dt {
			data = append(data, collect(nat.Attribute)...)
		}
	case *design.Array:
		data = append(data, collect(dt.ElemType)...)
	case *design.Map:
		data = append(data, collect(dt.KeyType)...)
		data = append(data, collect(dt.ElemType)...)
	}
	return
}

// buildRequestTypeData builds the type data and the constructor function
// for the server and client packages. It assumes that the gRPC request type
// is not nil.
//	* server side - initializes method payload type from the generated gRPC
//									request type in *.pb.go.
//	* client side - initializes generated gRPC request type in *.pb.go from
//									the method payload.
//
// svr param indicates that the type data is generated for server side.
func buildRequestTypeData(e *grpcdesign.EndpointExpr, sd *ServiceData, svr bool) *TypeData {
	buildInitFn := func(e *grpcdesign.EndpointExpr, sd *ServiceData, svr bool) *InitData {
		msgObj := design.AsObject(e.Request.Type)
		if len(*msgObj) == 0 || (svr && !needInit(e.MethodExpr.Payload.Type)) {
			return nil
		}
		var (
			name    string
			desc    string
			code    string
			retRef  string
			arg     *InitArgData
			srcPkg  string
			tgtPkg  string
			srcAtt  *design.AttributeExpr
			tgtAtt  *design.AttributeExpr
			cliArgs []*InitArgData

			svc    = sd.Service
			srcVar = "p"
			tgtVar = "v"
		)
		{
			if svr {
				name = "New" + svc.Scope.GoTypeName(e.MethodExpr.Payload)
				desc = fmt.Sprintf("%s builds the payload of the %q endpoint of the %q service from the gRPC request type.", name, e.Name(), svc.Name)
				srcAtt = e.Request
				tgtAtt = e.MethodExpr.Payload
				srcPkg = sd.PkgName
				tgtPkg = svc.PkgName
				retRef = svc.Scope.GoFullTypeRef(e.MethodExpr.Payload, svc.PkgName)
			} else {
				name = "New" + svc.Scope.GoTypeName(e.Request)
				desc = fmt.Sprintf("%s builds the gRPC request type from the payload of the %q endpoint of the %q service.", name, e.Name(), svc.Name)
				srcAtt = e.MethodExpr.Payload
				tgtAtt = e.Request
				srcPkg = svc.PkgName
				tgtPkg = sd.PkgName
				retRef = ProtoBufFullTypeRef(e.Request, sd.PkgName, svc.Scope)
			}
			code = protoBufTypeTransformHelper(srcAtt, tgtAtt, srcVar, tgtVar, srcPkg, tgtPkg, !svr, sd)
			arg = &InitArgData{
				Name:     srcVar,
				Ref:      srcVar,
				TypeName: svc.Scope.GoFullTypeName(srcAtt, srcPkg),
				TypeRef:  svc.Scope.GoFullTypeRef(srcAtt, srcPkg),
				Example:  srcAtt.Example(design.Root.API.Random()),
			}
		}
		return &InitData{
			Name:          name,
			Description:   desc,
			ReturnVarName: tgtVar,
			ReturnTypeRef: retRef,
			Code:          code,
			Args:          []*InitArgData{arg},
			CLIArgs:       cliArgs,
		}
	}

	var (
		name string
		ref  string

		svc = sd.Service
	)
	{
		name = ProtoBufMessageName(e.Request, svc.Scope)
		ref = ProtoBufFullTypeRef(e.Request, sd.PkgName, svc.Scope)
	}
	return &TypeData{
		Name: name,
		Ref:  ref,
		Init: buildInitFn(e, sd, svr),
	}
}

// buildResponseTypeData builds the type data and the constructor function
// for the server and client packages. It assumes that the gRPC response type
// is not nil.
//	* server side - initializes generated gRPC response type in *.pb.go from
//									the method result type.
//	* client side - initializes method result type from the generated gRPC
//									response type in *.pb.go from
//
// svr param indicates that the type data is generated for server side.
func buildResponseTypeData(e *grpcdesign.EndpointExpr, sd *ServiceData, svr bool) *TypeData {
	buildInitFn := func(e *grpcdesign.EndpointExpr, sd *ServiceData, svr bool) *InitData {
		msgObj := design.AsObject(e.Response.Message.Type)
		if len(*msgObj) == 0 || (!svr && !needInit(e.MethodExpr.Result.Type)) {
			return nil
		}
		var (
			name   string
			desc   string
			code   string
			retRef string
			arg    *InitArgData
			srcVar string
			srcPkg string
			tgtPkg string
			srcAtt *design.AttributeExpr
			tgtAtt *design.AttributeExpr

			svc    = sd.Service
			tgtVar = "v"
		)
		{
			if svr {
				name = "New" + svc.Scope.GoTypeName(e.Response.Message)
				desc = fmt.Sprintf("%s builds the gRPC response type from the result of the %q endpoint of the %q service.", name, e.Name(), svc.Name)
				srcVar = "res"
				srcAtt = e.MethodExpr.Result
				tgtAtt = e.Response.Message
				srcPkg = svc.PkgName
				tgtPkg = sd.PkgName
				retRef = ProtoBufFullTypeRef(e.Response.Message, sd.PkgName, svc.Scope)
			} else {
				name = "New" + svc.Scope.GoTypeName(e.MethodExpr.Result)
				desc = fmt.Sprintf("%s builds the result type of the %q endpoint of the %q service from the gRPC response type.", name, e.Name(), svc.Name)
				srcVar = "resp"
				srcAtt = e.Response.Message
				tgtAtt = e.MethodExpr.Result
				srcPkg = sd.PkgName
				tgtPkg = svc.PkgName
				retRef = svc.Scope.GoFullTypeRef(e.MethodExpr.Payload, svc.PkgName)
			}
			code = protoBufTypeTransformHelper(srcAtt, tgtAtt, srcVar, tgtVar, srcPkg, tgtPkg, svr, sd)
			arg = &InitArgData{
				Name:     srcVar,
				Ref:      srcVar,
				TypeName: svc.Scope.GoTypeName(srcAtt),
				TypeRef:  svc.Scope.GoFullTypeRef(srcAtt, srcPkg),
				Example:  srcAtt.Example(design.Root.API.Random()),
			}
		}
		return &InitData{
			Name:          name,
			Description:   desc,
			ReturnVarName: tgtVar,
			ReturnTypeRef: retRef,
			Code:          code,
			Args:          []*InitArgData{arg},
		}
	}

	var (
		name string
		ref  string

		svc = sd.Service
	)
	if svr {
		name = ProtoBufMessageName(e.Response.Message, svc.Scope)
		ref = ProtoBufFullTypeRef(e.Response.Message, sd.PkgName, svc.Scope)
	} else {
		name = svc.Scope.GoTypeName(e.MethodExpr.Result)
		ref = svc.Scope.GoFullTypeRef(e.MethodExpr.Result, svc.PkgName)
	}
	return &TypeData{
		Name: name,
		Ref:  ref,
		Init: buildInitFn(e, sd, svr),
	}
}

// buildErrorsData builds the error data for all the error responses in the
// endpoint expression. The response message for each error response are
// inferred from the method's error expression if not specified explicitly.
func buildErrorsData(e *grpcdesign.EndpointExpr, sd *ServiceData) []*ErrorData {
	var (
		errors []*ErrorData

		svc = sd.Service
	)
	errors = make([]*ErrorData, 0, len(e.GRPCErrors))
	for _, v := range e.GRPCErrors {
		var responseData *ResponseData
		{
			responseData = &ResponseData{
				StatusCode:  statusCodeToGRPCConst(v.Response.StatusCode),
				Description: v.Response.Description,
			}
		}
		errors = append(errors, &ErrorData{
			Name:     v.Name,
			Ref:      svc.Scope.GoFullTypeRef(v.ErrorExpr.AttributeExpr, svc.PkgName),
			Response: responseData,
		})
	}
	return errors
}

// protoBufTypeTransformHelper is a helper function to transform a protocol
// buffer message type to a Go type and vice versa. If src and tgt are of
// different types (i.e. the Payload/Result is a non-user type and
// Request/Response message is always a user type), the function returns the
// code for initializing the types appropriately by making use of the wrapped
// "field" attribute. Use this function in places where
// codegen.ProtoBufTypeTransform needs to be called.
func protoBufTypeTransformHelper(src, tgt *design.AttributeExpr, srcVar, tgtVar, srcPkg, tgtPkg string, proto bool, sd *ServiceData) string {
	var (
		code string
		err  error
		h    []*codegen.TransformFunctionData

		svc = sd.Service
	)
	if e := isCompatible(src.Type, tgt.Type, srcVar, tgtVar); e == nil {
		code, h, err = ProtoBufTypeTransform(src.Type, tgt.Type, srcVar, tgtVar, srcPkg, tgtPkg, proto, svc.Scope)
		if err != nil {
			fmt.Println(err.Error()) // TBD validate DSL so errors are not possible
			return ""
		}
		sd.TransformHelpers = codegen.AppendHelpers(sd.TransformHelpers, h)
		return code
	}
	if proto {
		// tgt is a protocol buffer message type. src type is wrapped in an
		// attribute called "field" in tgt.
		pbType := ProtoBufFullMessageName(tgt, tgtPkg, svc.Scope)
		code = fmt.Sprintf("%s := &%s{\nField: %s,\n}", tgtVar, pbType, typeConvert(srcVar, src.Type, tgt.Type, proto))
	} else {
		// tgt is a Go type. src is a protocol buffer message type.
		code = fmt.Sprintf("%s := %s\n", tgtVar, typeConvert(srcVar+".Field", src.Type, tgt.Type, proto))
	}
	return code
}

// needInit returns true if and only if the given type is or makes use of user
// types.
func needInit(dt design.DataType) bool {
	if dt == design.Empty {
		return false
	}
	switch actual := dt.(type) {
	case design.Primitive:
		return false
	case *design.Array:
		return needInit(actual.ElemType.Type)
	case *design.Map:
		return needInit(actual.KeyType.Type) ||
			needInit(actual.ElemType.Type)
	case *design.Object:
		for _, nat := range *actual {
			if needInit(nat.Attribute.Type) {
				return true
			}
		}
		return false
	case design.UserType:
		return true
	default:
		panic(fmt.Sprintf("unknown data type %T", actual)) // bug
	}
}

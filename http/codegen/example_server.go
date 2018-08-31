package codegen

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

// ExampleServerFiles returns and example main and dummy service
// implementations.
func ExampleServerFiles(genpkg string, root *expr.RootExpr) []*codegen.File {
	var fw []*codegen.File
	if m := exampleServer(genpkg, root); m != nil {
		fw = append(fw, m)
	}
	if f := dummyMultipart(genpkg, root); f != nil {
		fw = append(fw, f)
	}
	return fw
}

func exampleServer(genpkg string, root *expr.RootExpr) *codegen.File {
	mainPath := filepath.Join("cmd", codegen.SnakeCase(codegen.Goify(root.API.Name, true))+"_svc", "http.go")
	if _, err := os.Stat(mainPath); !os.IsNotExist(err) {
		return nil // file already exists, skip it.
	}
	idx := strings.LastIndex(genpkg, string(os.PathSeparator))
	rootPath := "."
	if idx > 0 {
		rootPath = genpkg[:idx]
	}
	apiPkg := strings.ToLower(codegen.Goify(root.API.Name, false))
	specs := []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "log"},
		{Path: "net/http"},
		{Path: "os"},
		{Path: "time"},
		{Path: "goa.design/goa/http", Name: "goahttp"},
		{Path: "goa.design/goa/http/middleware"},
		{Path: "github.com/gorilla/websocket"},
		{Path: rootPath, Name: apiPkg},
	}
	for _, svc := range root.HTTPServices {
		pkgName := HTTPServices.Get(svc.Name()).Service.PkgName
		specs = append(specs, &codegen.ImportSpec{
			Path: path.Join(genpkg, "http", codegen.SnakeCase(svc.Name()), "server"),
			Name: pkgName + "svr",
		})
		specs = append(specs, &codegen.ImportSpec{
			Path: path.Join(genpkg, codegen.SnakeCase(svc.Name())),
			Name: pkgName,
		})
	}
	sections := []*codegen.SectionTemplate{codegen.Header("", "main", specs)}
	svcdata := make([]*ServiceData, 0, len(root.HTTPServices))
	for _, svc := range root.HTTPServices {
		svcdata = append(svcdata, HTTPServices.Get(svc.Name()))
	}
	if needStream(svcdata) {
		specs = append(specs, &codegen.ImportSpec{Path: "github.com/gorilla/websocket"})
	}
	data := map[string]interface{}{
		"Services": svcdata,
		"APIPkg":   apiPkg,
	}
	sections = append(sections, &codegen.SectionTemplate{
		Name:   "serve-http",
		Source: serveHTTPT,
		Data:   data,
		FuncMap: map[string]interface{}{
			"needStream": needStream,
		},
	})

	return &codegen.File{Path: mainPath, SectionTemplates: sections}
}

// dummyMultipart returns a dummy implementation of the multipart decoders
// and encoders.
func dummyMultipart(genpkg string, root *expr.RootExpr) *codegen.File {
	mpath := "multipart.go"
	if _, err := os.Stat(mpath); !os.IsNotExist(err) {
		return nil // file already exists, skip it.
	}
	var (
		sections []*codegen.SectionTemplate
		mustGen  bool

		apiPkg = strings.ToLower(codegen.Goify(root.API.Name, false))
	)
	{
		specs := []*codegen.ImportSpec{
			{Path: "mime/multipart"},
		}
		for _, svc := range root.HTTPServices {
			pkgName := HTTPServices.Get(svc.Name()).Service.PkgName
			specs = append(specs, &codegen.ImportSpec{
				Path: path.Join(genpkg, codegen.SnakeCase(svc.Name())),
				Name: pkgName,
			})
		}
		header := codegen.Header("", apiPkg, specs)
		sections = []*codegen.SectionTemplate{header}
		for _, svc := range root.HTTPServices {
			data := HTTPServices.Get(svc.Name())
			for _, e := range data.Endpoints {
				if e.MultipartRequestDecoder != nil {
					mustGen = true
					sections = append(sections, &codegen.SectionTemplate{
						Name:   "dummy-multipart-request-decoder",
						Source: dummyMultipartRequestDecoderImplT,
						Data:   e.MultipartRequestDecoder,
					})
				}
				if e.MultipartRequestEncoder != nil {
					mustGen = true
					sections = append(sections, &codegen.SectionTemplate{
						Name:   "dummy-multipart-request-encoder",
						Source: dummyMultipartRequestEncoderImplT,
						Data:   e.MultipartRequestEncoder,
					})
				}
			}
		}
	}
	if !mustGen {
		return nil
	}
	return &codegen.File{
		Path:             mpath,
		SectionTemplates: sections,
	}
}

// needStream returns true if at least one method in the list of services
// uses stream for sending payload/result.
func needStream(data []*ServiceData) bool {
	for _, svc := range data {
		if streamingEndpointExists(svc) {
			return true
		}
	}
	return false
}

// input: MultipartData
const dummyMultipartRequestDecoderImplT = `{{ printf "%s implements the multipart decoder for service %q endpoint %q. The decoder must populate the argument p after encoding." .FuncName .ServiceName .MethodName | comment }}
func {{ .FuncName }}(mr *multipart.Reader, p *{{ .Payload.Ref }}) error {
	// Add multipart request decoder logic here
	return nil
}
`

// input: MultipartData
const dummyMultipartRequestEncoderImplT = `{{ printf "%s implements the multipart encoder for service %q endpoint %q." .FuncName .ServiceName .MethodName | comment }}
func {{ .FuncName }}(mw *multipart.Writer, p {{ .Payload.Ref }}) error {
	// Add multipart request encoder logic here
	return nil
}
`

// input: map[string]interface{}{"Services":[]ServiceData, "APIPkg": string}
const serveHTTPT = `func httpServe(addr string{{ range .Services }}{{ if .Endpoints }}, {{ .Service.VarName }}Endpoints *{{ .Service.PkgName }}.Endpoints{{ end }}{{ end }}, errc chan error, logger *log.Logger, debug bool) *http.Server {
	// Setup goa log adapter. Replace logger with your own using your
	// log package of choice. The goa.design/middleware/logging/...
	// ackages define log adapters for common log packages.
	var (
		adapter middleware.Logger
	)
	{
		adapter = middleware.NewLogger(logger)
	}

	// Provide the transport specific request decoder and response encoder.
	// The goa http package has built-in support for JSON, XML and gob.
	// Other encodings can be used by providing the corresponding functions,
	// see goa.design/encoding.
	var (
		dec = goahttp.RequestDecoder
		enc = goahttp.ResponseEncoder
	)

	// Build the service HTTP request multiplexer and configure it to serve
	// HTTP requests to the service endpoints.
	var mux goahttp.Muxer
	{
		mux = goahttp.NewMuxer()
	}

	// Wrap the endpoints with the transport specific layers. The generated
	// server packages contains code generated from the design which maps
	// the service input and output data structures to HTTP requests and
	// responses.
	var (
	{{- range .Services }}
		{{ .Service.VarName }}Server *{{.Service.PkgName}}svr.Server
	{{- end }}
	)
	{
		eh := errorHandler(logger)
	{{- if needStream .Services }}
		upgrader := &websocket.Upgrader{}
	{{- end }}
	{{- range .Services }}
		{{- if .Endpoints }}
		{{ .Service.VarName }}Server = {{ .Service.PkgName }}svr.New({{ .Service.VarName }}Endpoints, mux, dec, enc, eh{{ if needStream $.Services }}, upgrader, nil{{ end }}{{ range .Endpoints }}{{ if .MultipartRequestDecoder }}, {{ $.APIPkg }}.{{ .MultipartRequestDecoder.FuncName }}{{ end }}{{ end }})
		{{-  else }}
		{{ .Service.VarName }}Server = {{ .Service.PkgName }}svr.New(nil, mux, dec, enc, eh)
		{{-  end }}
	{{- end }}
	}

	// Configure the mux.
	{{- range .Services }}
	{{ .Service.PkgName }}svr.Mount(mux{{ if .Endpoints }}, {{ .Service.VarName }}Server{{ end }})
	{{- end }}

	// Wrap the multiplexer with additional middlewares. Middlewares mounted
	// here apply to all the service endpoints.
	var handler http.Handler = mux
	{
		if debug {
			handler = middleware.Debug(mux, os.Stdout)(handler)
		}
		handler = middleware.Log(adapter)(handler)
		handler = middleware.RequestID()(handler)
	}

	// Start HTTP server using default configuration, change the code to
	// configure the server as required by your service.
	srv := &http.Server{Addr: addr, Handler: handler}
	go func() {
		{{- range .Services }}
		for _, m := range {{ .Service.VarName }}Server.Mounts {
			{{- if .FileServers }}
			logger.Printf("file %q mounted on %s %s", m.Method, m.Verb, m.Pattern)
			{{- else }}
			logger.Printf("method %q mounted on %s %s", m.Method, m.Verb, m.Pattern)
			{{- end }}
		}
		{{- end }}
		logger.Printf("HTTP listening on %s", addr)
		errc <- srv.ListenAndServe()
	}()

	return srv
}

func httpStop(srv *http.Server) {
	// Shutdown gracefully with a 30s timeout.
  ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
  defer cancel()
	srv.Shutdown(ctx)
}

// errorHandler returns a function that writes and logs the given error.
// The function also writes and logs the error unique ID so that it's possible
// to correlate.
func errorHandler(logger *log.Logger) func(context.Context, http.ResponseWriter, error) {
	return func(ctx context.Context, w http.ResponseWriter, err error) {
		id := ctx.Value(middleware.RequestIDKey).(string)
		w.Write([]byte("[" + id + "] encoding: " + err.Error()))
		logger.Printf("[%s] ERROR: %s", id, err.Error())
	}
}
`

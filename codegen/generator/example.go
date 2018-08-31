package generator

import (
	"goa.design/goa/codegen"
	"goa.design/goa/codegen/service"
	"goa.design/goa/eval"
	"goa.design/goa/expr"
	grpccodegen "goa.design/goa/grpc/codegen"
	httpcodegen "goa.design/goa/http/codegen"
)

// Example iterates through the roots and returns files that implement an
// example service and client.
func Example(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
	var (
		files      []*codegen.File
		transports []*service.TransportData
	)
	for _, root := range roots {
		r, ok := root.(*expr.RootExpr)
		if !ok {
			continue // could be a plugin root expression
		}

		// Auth
		f := service.AuthFuncsFile(genpkg, r)
		if f != nil {
			files = append(files, f)
		}

		// HTTP
		if len(r.HTTPServices) > 0 {
			svcs := make([]string, 0, len(r.HTTPServices))
			for _, s := range r.HTTPServices {
				svcs = append(svcs, s.Name())
			}
			transports = append(transports, &service.TransportData{
				Name:        "http",
				DisplayName: "HTTP",
				Services:    svcs,
				Host:        "http://localhost",
				Port:        "8080",
			})
			files = append(files, httpcodegen.ExampleServerFiles(genpkg, r)...)
			if cli := httpcodegen.ExampleCLI(genpkg, r); cli != nil {
				files = append(files, cli)
			}
		}

		// GRPC
		if len(r.GRPCServices) > 0 {
			svcs := make([]string, 0, len(r.GRPCServices))
			for _, s := range r.GRPCServices {
				svcs = append(svcs, s.Name())
			}
			transports = append(transports, &service.TransportData{
				Name:        "grpc",
				DisplayName: "gRPC",
				Services:    svcs,
				Host:        "http://localhost",
				Port:        "8081",
			})
			if f := grpccodegen.ExampleServerFiles(genpkg, r); f != nil {
				files = append(files, f)
			}
			if cli := grpccodegen.ExampleCLI(genpkg, r); cli != nil {
				files = append(files, cli)
			}
		}

		// server main
		if fs := service.ExampleServiceFiles(genpkg, r, transports); len(fs) != 0 {
			files = append(files, fs...)
		}

		// client main
		if f := service.ExampleCLI(genpkg, r, transports); f != nil {
			files = append(files, f)
		}
	}

	// Set a default transport. If both HTTP and gRPC transports are
	// available, set HTTP as default else set the only available transport
	// as default.
	tlen := len(transports)
	switch {
	case tlen == 0:
		panic("no transports available!")
	case tlen > 1:
		for _, t := range transports {
			if t.Name == "http" {
				t.IsDefault = true
			}
		}
	case tlen == 1:
		transports[0].IsDefault = true
		// If there is only one transport, we can start the service using
		// port :8080 by default.
		transports[0].Port = "8080"
	}

	return files, nil
}

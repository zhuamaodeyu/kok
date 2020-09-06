package chi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/RussellLuo/kok/gen/util/generator"
	"github.com/RussellLuo/kok/gen/util/misc"
	"github.com/RussellLuo/kok/gen/util/openapi"
	"github.com/RussellLuo/kok/gen/util/reflector"
)

var (
	template = `// Code generated by kok; DO NOT EDIT.
// github.com/RussellLuo/kok

{{- $pkgName := .Result.PkgName}}
{{- $enableTracing := .Opts.EnableTracing}}

package {{$pkgName}}

import (
	"encoding/json"
	"net/http"
	"strconv"
	"github.com/go-chi/chi"
	{{- if $enableTracing}}
	"github.com/RussellLuo/kok/pkg/trace/xnet"
	{{- end}}
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/kit/endpoint"
	httpcodec "github.com/RussellLuo/kok/pkg/codec/httpv2"
	"github.com/RussellLuo/kok/pkg/oasv2"

	{{- range .Result.Imports}}
	"{{.}}"
	{{- end}}
)

func NewHTTPRouter(svc {{.Result.SrcPkgPrefix}}{{.Result.Interface.Name}}, codecs httpcodec.Codecs) chi.Router {
	return NewHTTPRouterWithOAS(svc, codecs, nil)
}

func NewHTTPRouterWithOAS(svc {{.Result.SrcPkgPrefix}}{{.Result.Interface.Name}}, codecs httpcodec.Codecs, schema oasv2.Schema) chi.Router {
	r := chi.NewRouter()

	{{if $enableTracing -}}
	contextor := xnet.NewContextor()
	r.Method("PUT", "/trace", xnet.HTTPHandler(contextor))
	{{- end}}

	if schema != nil {
		r.Method("GET", "/api", oasv2.Handler(OASv2APIDoc, schema))
	}

	var codec httpcodec.Codec
	var options []kithttp.ServerOption

	{{- range .Spec.Operations}}

	codec = codecs.EncodeDecoder("{{.Name}}")
	r.Method(
		"{{.Method}}", "{{.Pattern}}",
		kithttp.NewServer(
			MakeEndpointOf{{.Name}}(svc),
			decode{{.Name}}Request(codec),
			httpcodec.MakeResponseEncoder(codec, {{getStatusCode .SuccessResponse.StatusCode .Name}}),
			append(options,
				kithttp.ServerErrorEncoder(httpcodec.MakeErrorEncoder(codec)),
				{{- if $enableTracing}}
				kithttp.ServerBefore(contextor.HTTPToContext("{{$pkgName}}", "{{.Name}}")),
				{{- end}}
			)...,
		),
	)
	{{- end}}

	return r
}

{{- range .Spec.Operations}}

{{- $nonCtxParams := nonCtxParams .Request.Params}}
{{- $nonBodyParams := nonBodyParams $nonCtxParams}}
{{- $bodyParams := bodyParams $nonCtxParams}}

func decode{{.Name}}Request(codec httpcodec.Codec) kithttp.DecodeRequestFunc {
	return func(_ context.Context, r *http.Request) (interface{}, error) {
		{{- if $nonCtxParams}}
		var req {{.Name}}Request

		{{end -}}

		{{if $bodyParams -}}
		if err := codec.DecodeRequestBody(r.Body, &req); err != nil {
			return nil, err
		}
		{{end -}}

		{{- range $nonBodyParams}}

		{{- if .Sub}} {{/* This is a parent parameter */}}
		{{- $parentName := .Name}}

		{{- range .Sub}}
		{{lowerFirst .Name}} := {{extractParam .}}
		if err := codec.DecodeRequestParam("{{$parentName}}.{{.Name}}", {{lowerFirst .Name}}, &req.{{title $parentName}}.{{.Name}}); err != nil {
			return nil, err
		}

		{{end -}} {{/* End of range .Sub */}}

		{{- else}} {{/* This is a normal (non-parent) parameter */}}

		{{.Name}} := {{extractParam .}}
		if err := codec.DecodeRequestParam("{{.Name}}", {{.Name}}, &req.{{title .Name}}); err != nil {
			return nil, err
		}
		{{- end}} {{/* End of if .Sub */}}

		{{end -}} {{/* End of range $nonBodyParams */}}

		{{- if $nonCtxParams}}

		return {{addAmpersand "req"}}, nil
		{{- else -}}
		return nil, nil
		{{- end}} {{/* End of if $nonCtxParams */}}
	}
}

{{- end}}
`
)

type RequestField struct {
	Name  string
	Value string
}

type Server struct {
	Service     interface{}
	NewEndpoint interface{}
	Request     interface{}
	Response    interface{}
}

type Options struct {
	SchemaPtr         bool
	SchemaTag         string
	TagKeyToSnakeCase bool
	Formatted         bool
	EnableTracing     bool
}

type Generator struct {
	opts *Options
}

func New(opts *Options) *Generator {
	return &Generator{opts: opts}
}

func (g *Generator) Generate(result *reflector.Result, spec *openapi.Specification) ([]byte, error) {
	data := struct {
		Result *reflector.Result
		Spec   *openapi.Specification
		Opts   *Options
	}{
		Result: result,
		Spec:   spec,
		Opts:   g.opts,
	}

	methodMap := make(map[string]*reflector.Method)
	for _, method := range result.Interface.Methods {
		methodMap[method.Name] = method
	}

	return generator.Generate(template, data, generator.Options{
		Funcs: map[string]interface{}{
			"title":      strings.Title,
			"lowerFirst": misc.LowerFirst,
			"addAmpersand": func(name string) string {
				if g.opts.SchemaPtr {
					return "&" + name
				}
				return name
			},
			"extractParam": func(param *openapi.Param) string {
				switch param.In {
				case openapi.InPath:
					return fmt.Sprintf(`chi.URLParam(r, "%s")`, param.Alias)
				case openapi.InQuery:
					return fmt.Sprintf(`r.URL.Query().Get("%s")`, param.Alias)
				case openapi.InHeader:
					return fmt.Sprintf(`r.Header.Get("%s")`, param.Alias)
				default:
					panic(fmt.Errorf("param.In `%s` not supported", param.In))
				}
			},
			"nonBodyParams": func(in []*openapi.Param) (out []*openapi.Param) {
				for _, p := range in {
					if p.In != openapi.InBody {
						out = append(out, p)
					}
				}
				return
			},
			"bodyParams": func(in []*openapi.Param) (out []*openapi.Param) {
				for _, p := range in {
					if p.In == openapi.InBody {
						out = append(out, p)
					}
				}
				return
			},
			"nonCtxParams": func(params []*openapi.Param) (out []*openapi.Param) {
				for _, p := range params {
					if p.Type != "context.Context" {
						out = append(out, p)
					}
				}
				return
			},
			"getStatusCode": func(givenStatusCode int, name string) int {
				method, ok := methodMap[name]
				if !ok {
					panic(fmt.Errorf("no method named %q", name))
				}

				if len(method.Returns) > 0 {
					// Use the given status code, since the corresponding
					// method is a fruitful function.
					return givenStatusCode
				}

				if givenStatusCode == http.StatusOK {
					fmt.Printf("NOTE: statusCode is changed to be 204, since method %q returns no result\n", name)
					return http.StatusNoContent
				}

				if givenStatusCode != http.StatusNoContent {
					panic(fmt.Errorf("statusCode must be 204, since method %q returns no result", name))
				}
				return givenStatusCode
			},
		},
		Formatted: g.opts.Formatted,
	})
}

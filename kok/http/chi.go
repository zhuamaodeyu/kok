package http

import (
	"errors"
	"fmt"
	"strings"

	"github.com/RussellLuo/kok/kok/endpoint"
	"github.com/RussellLuo/kok/kok/gen"
	"github.com/RussellLuo/kok/oapi"
	"github.com/RussellLuo/kok/reflector"
)

var (
	template = `// Code generated by kok; DO NOT EDIT.
// github.com/RussellLuo/kok

package {{.Result.PkgName}}

import (
	"encoding/json"
	"net/http"
	"strconv"
	"github.com/go-chi/chi"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/kit/endpoint"
	{{- range .Result.Imports }}
	"{{.}}"
	{{- end }}
)


func NewHTTPHandler(svc {{.Result.SrcPkgPrefix}}{{.Result.Interface.Name}}) chi.Router {
	r := chi.NewRouter()

	options := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(errorEncoder),
	}
	{{range .Spec.Operations}}
	r.Method(
		"{{.Method}}", "{{.Pattern}}",
		kithttp.NewServer(
			MakeEndpointOf{{.Name}}(svc),
			decode{{.Name}}Request,
			encodeGenericResponse,
			options...,
		),
	)
	{{- end}}

	return r
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	// err2code (signature: func(error) int) must be provided in this package,
	// to transform a business error to an HTTP code!
	w.WriteHeader(err2code(err))
	json.NewEncoder(w).Encode(errorWrapper{Error: err.Error()})
}

type errorWrapper struct {
	Error string ` + "`" + `json:"error"` + "`" + `
}

{{- range .Spec.Operations}}

{{- $nonCtxParams := nonCtxParams .Request.Params}}

func decode{{.Name}}Request(_ context.Context, r *http.Request) (interface{}, error) {
	{{$nonBodyParams := nonBodyParams $nonCtxParams -}}
	{{range $nonBodyParams -}}

	{{- if eq .Type "string" -}}
	{{.Name}} := {{extractParam .}}
	{{- else -}}
	{{.Name}}Value := {{extractParam .}}
	{{.Name}}, err := {{parseExpr .Name .Type}}
	if err != nil {
		return nil, err
	}
	{{end}}

	{{end -}}

	{{- $bodyParams := bodyParams $nonCtxParams}}
	{{- if $bodyParams -}}
	var body struct {
		{{- range $bodyParams}}
		{{title .Name}} {{.Type}} {{addTag .Name .Type}}
		{{- end}}
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}
	{{- end}}

	return {{addAmpersand .Name}}Request{
		{{- range $nonCtxParams}}

		{{- if eq .In "body"}}
		{{title .Name}}: body.{{title .Name}},
		{{- else}}
		{{title .Name}}: {{castIfInt .Name .Type}},
		{{- end}}

		{{- end}}
	}, nil
}

{{- end}}

func encodeGenericResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if f, ok := response.(endpoint.Failer); ok && f.Failed() != nil {
		errorEncoder(ctx, f.Failed(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
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
}

type ChiGenerator struct {
	opts Options
}

func NewChi(opts Options) *ChiGenerator {
	return &ChiGenerator{opts: opts}
}

func (c *ChiGenerator) Generate(result *reflector.Result, spec *oapi.Specification) ([]byte, error) {
	data := struct {
		Result *reflector.Result
		Spec   *oapi.Specification
	}{
		Result: result,
		Spec:   spec,
	}

	return gen.Generate(template, data, gen.Options{
		Funcs: map[string]interface{}{
			"title": strings.Title,
			"addTag": func(name, typ string) string {
				if c.opts.SchemaTag == "" {
					return ""
				}

				if typ == "error" {
					name = "-"
				} else if c.opts.TagKeyToSnakeCase {
					name = endpoint.ToSnakeCase(name)
				}

				return fmt.Sprintf("`%s:\"%s\"`", c.opts.SchemaTag, name)
			},
			"addAmpersand": func(name string) string {
				if c.opts.SchemaPtr {
					return "&" + name
				}
				return name
			},
			"extractParam": func(param *oapi.Param) string {
				switch param.In {
				case oapi.InPath:
					return fmt.Sprintf(`chi.URLParam(r, "%s")`, param.Name)
				case oapi.InQuery:
					return fmt.Sprintf(`r.URL.Query().Get("%s")`, param.Name)
				default:
					panic(errors.New(fmt.Sprintf("param.In `%s` not supported", param.In)))
				}
			},
			"nonBodyParams": func(in []*oapi.Param) (out []*oapi.Param) {
				for _, p := range in {
					if p.In != oapi.InBody {
						out = append(out, p)
					}
				}
				return
			},
			"bodyParams": func(in []*oapi.Param) (out []*oapi.Param) {
				for _, p := range in {
					if p.In == oapi.InBody {
						out = append(out, p)
					}
				}
				return
			},
			"nonCtxParams": func(params []*oapi.Param) (out []*oapi.Param) {
				for _, p := range params {
					if p.Type != "context.Context" {
						out = append(out, p)
					}
				}
				return
			},
			"parseExpr": func(name, typ string) string {
				switch typ {
				case "int", "int8", "int16", "int32", "int64":
					return fmt.Sprintf("strconv.ParseInt(%sValue, 10, 64)", name)
				case "uint", "uint8", "uint16", "uint32", "uint64":
					return fmt.Sprintf("strconv.ParseUint(%sValue, 10, 64)", name)
				default:
					panic(fmt.Errorf("unrecognized integer type %s", typ))
				}
			},
			"castIfInt": func(name, typ string) string {
				switch typ {
				case "int", "int8", "int16", "int32",
					"uint", "uint8", "uint16", "uint32":
					return fmt.Sprintf("%s(%s)", typ, name)
				default:
					return name
				}
			},
		},
		Formatters: []gen.Formatter{gen.Gofmt, gen.Goimports},
	})
}
package optizz

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"strings"
	"time"

	"github.com/wI2L/fizz/openapi"
)

const ctxOpenAPIOperation = "_ctx_openapi_operation"

// Primitive type helpers.
var (
	Integer  int32
	Long     int64
	Float    float32
	Double   float64
	String   string
	Byte     []byte
	Binary   []byte
	Boolean  bool
	DateTime time.Time
)

// Fizz is an abstraction of a Fiber app that wraps the
// routes handlers with Tonic and generates an OpenAPI
// 3.0 specification from it.
type Optizz struct {
	gen *openapi.Generator
	app *fiber.App
	*RouterGroup
}


// New creates a new Fizz wrapper for
// a default Fiber app.
func New() *Optizz {
	return NewFromApp(fiber.New())
}

// NewFromApp creates a new Fizz wrapper
// from an existing Fiber app.
func NewFromApp(app *fiber.App) *Optizz {
	// Create a new spec with the config
	// based on tonic internals.
	gen, _ := openapi.NewGenerator(
		&openapi.SpecGenConfig{
			ValidatorTag:      ValidationTag,
			PathLocationTag:   PathTag,
			QueryLocationTag:  QueryTag,
			HeaderLocationTag: HeaderTag,
			EnumTag:           EnumTag,
			DefaultTag:        DefaultTag,
		},
	)

	return &Optizz{
		app: app,
		gen: gen,
		RouterGroup: &RouterGroup{
			app:   app,
			group: app.Group(""),
			gen:   gen,
			path:  "",
		},
	}
}

// App returns the underlying Fiber app.
func (f *Optizz) App() *fiber.App {
	return f.app
}

// Generator returns the underlying OpenAPI generator.
func (f *Optizz) Generator() *openapi.Generator {
	return f.gen
}

// Errors returns the errors that may have occurred
// during the spec generation.
func (f *Optizz) Errors() []error {
	return f.gen.Errors()
}


// OpenAPI returns a Fiber HandlerFunc that serves
// the marshalled OpenAPI specification of the API.
func (f *Optizz) OpenAPI(info *openapi.Info, ct string) fiber.Handler {
	f.gen.SetInfo(info)

	ct = strings.ToLower(ct)
	if ct == "" {
		ct = "json"
	}
	switch ct {
	case "json":
		return func(c *fiber.Ctx) error {
			c.Status(200)
			c.JSON(f.gen.API())
			return nil
		}
		//case "yaml":
		//	return func(c *fiber.Ctx) error {
		//		c.YAML(200, f.gen.API())
		//	}
	}
	panic("invalid content type, use JSON or YAML")
}

// OperationOption represents an option-pattern function
// used to add informations to an operation.
type OperationOption func(*openapi.OperationInfo)

// StatusDescription sets the default status description of the operation.
func StatusDescription(desc string) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.StatusDescription = desc
	}
}

// Summary adds a summary to an operation.
func Summary(summary string) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Summary = summary
	}
}

// Summaryf adds a summary to an operation according
// to a format specifier.
func Summaryf(format string, a ...interface{}) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Summary = fmt.Sprintf(format, a...)
	}
}

// Description adds a description to an operation.
func Description(desc string) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Description = desc
	}
}

// Descriptionf adds a description to an operation
// according to a format specifier.
func Descriptionf(format string, a ...interface{}) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Description = fmt.Sprintf(format, a...)
	}
}

// ID overrides the operation ID.
func ID(id string) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.ID = id
	}
}

// Deprecated marks the operation as deprecated.
func Deprecated(deprecated bool) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Deprecated = deprecated
	}
}

// Response adds an additional response to the operation.
func Response(statusCode, desc string, model interface{}, headers []*openapi.ResponseHeader, example interface{}) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Responses = append(o.Responses, &openapi.OperationResponse{
			Code:        statusCode,
			Description: desc,
			Model:       model,
			Headers:     headers,
			Example:     example,
		})
	}
}

// ResponseWithExamples is a variant of Response that accept many examples.
func ResponseWithExamples(statusCode, desc string, model interface{}, headers []*openapi.ResponseHeader, examples map[string]interface{}) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Responses = append(o.Responses, &openapi.OperationResponse{
			Code:        statusCode,
			Description: desc,
			Model:       model,
			Headers:     headers,
			Examples:    examples,
		})
	}
}

// Header adds a header to the operation.
func Header(name, desc string, model interface{}) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.Headers = append(o.Headers, &openapi.ResponseHeader{
			Name:        name,
			Description: desc,
			Model:       model,
		})
	}
}

// InputModel overrides the binding model of the operation.
func InputModel(model interface{}) func(*openapi.OperationInfo) {
	return func(o *openapi.OperationInfo) {
		o.InputModel = model
	}
}

// OperationFromContext returns the OpenAPI operation from
// the given Fiber context or an error if none is found.
func OperationFromContext(c *fiber.Ctx) (*openapi.Operation, error) {
	if v := c.Locals(ctxOpenAPIOperation); v != nil {
		if op, ok := v.(*openapi.Operation); ok {
			return op, nil
		}
		return nil, errors.New("invalid type: not an operation")
	}
	return nil, errors.New("operation not found")
}
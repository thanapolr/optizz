package optizz

import (
	"encoding"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"gopkg.in/go-playground/validator.v9"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Fields tags used by optic.
const (
	QueryTag      = "query"
	PathTag       = "path"
	HeaderTag     = "header"
	EnumTag       = "enum"
	RequiredTag   = "required"
	DefaultTag    = "default"
	ValidationTag = "validate"
	ExplodeTag    = "explode"
)

var (
	errorHook      ErrorHook  = DefaultErrorHook
	bindHook       BindHook   = DefaultBindingHook
	bindQueryHook  BindHook   = DefaultBindQueryHook
	bindPathHook   BindHook   = DefaultBindPathHook
	bindHeaderHook BindHook   = DefaultBindHeaderHook
	renderHook     RenderHook = DefaultRenderHook
	execHook       ExecHook   = DefaultExecHook
)

// BindHook is the hook called by the wrapping Fiber-handler when
// binding an incoming request to the optic-handler's input object.
type BindHook func(*fiber.Ctx, reflect.Value) error

// RenderHook is the last hook called by the wrapping Fiber-handler
// before returning. It takes the Fiber context, the HTTP status code
// and the response payload as parameters.
// Its role is to render the payload to the client to the
// proper format.
type RenderHook func(*fiber.Ctx, int, interface{})

// ErrorHook lets you interpret errors returned by your handlers.
// After analysis, the hook should return a suitable http status code
// and and error payload.
// This lets you deeply inspect custom error types.
type ErrorHook func(*fiber.Ctx, error) (int, interface{})

// An ExecHook is the func called to handle a request.
// The default ExecHook simply calle the wrapping Fiber-handler
// with the Fiber context.
type ExecHook func(*fiber.Ctx, fiber.Handler, string) error

// DefaultErrorHook is the default error hook.
// It returns a StatusBadRequest with a payload containing
// the error message.
func DefaultErrorHook(c *fiber.Ctx, e error) (int, interface{}) {
	return http.StatusBadRequest, map[string]string{
		"error": e.Error(),
	}
}

// DefaultBindingHook is the default binding hook.
// It uses Fiber JSON binding to bind the body parameters of the request
// to the input object of the handler.
// It returns an error if Fiber binding fails.
func DefaultBindingHook(c *fiber.Ctx, v reflect.Value) error {
	i := v.Interface()
	if c.Method() == http.MethodGet || c.Request().Header.ContentLength() == 0 {
		return nil
	}

	if err := c.BodyParser(i); err != nil && err != io.EOF {
		return fmt.Errorf("error parsing request body: %s", err.Error())
	}
	return nil
}

func DefaultBindQueryHook(c *fiber.Ctx, v reflect.Value) error {
	return bind(c, v, QueryTag, extractQuery)
}

func DefaultBindPathHook(c *fiber.Ctx, v reflect.Value) error {
	return bind(c, v, PathTag, extractPath)
}

func DefaultBindHeaderHook(c *fiber.Ctx, v reflect.Value) error {
	return bind(c, v, HeaderTag, extractHeader)
}

// DefaultRenderHook is the default render hook.
// It marshals the payload to JSON, or returns an empty body if the payload is nil.
func DefaultRenderHook(c *fiber.Ctx, statusCode int, payload interface{}) {
	if payload != nil {
		c.Status(statusCode).JSON(payload)
	} else {
		c.Status(statusCode).Format("")
	}
}

// DefaultExecHook is the default exec hook.
// It simply executes the wrapping Fiber-handler with
// the given context.
func DefaultExecHook(c *fiber.Ctx, h fiber.Handler, fname string) error {
	return h(c)
}

// GetErrorHook returns the current error hook.
func GetErrorHook() ErrorHook {
	return errorHook
}

// SetErrorHook sets the given hook as the
// default error handling hook.
func SetErrorHook(eh ErrorHook) {
	if eh != nil {
		errorHook = eh
	}
}

// GetBindHook returns the current bind hook.
func GetBindHook() BindHook {
	return bindHook
}

// SetBindHook sets the given hook as the
// default binding hook.
func SetBindHook(bh BindHook) {
	if bh != nil {
		bindHook = bh
	}
}

// GetBindQueryHook returns the current bind hook.
func GetBindQueryHook() BindHook {
	return bindQueryHook
}

// SetBindQueryHook sets the given hook as the
// default binding hook.
func SetBindQueryHook(bh BindHook) {
	if bh != nil {
		bindQueryHook = bh
	}
}

// GetBindQueryHook returns the current bind hook.
func GetBindPathHook() BindHook {
	return bindPathHook
}

// SetBindQueryHook sets the given hook as the
// default binding hook.
func SetBindPathHook(bh BindHook) {
	if bh != nil {
		bindPathHook = bh
	}
}

// GetBindQueryHook returns the current bind hook.
func GetBindHeaderHook() BindHook {
	return bindHeaderHook
}

// SetBindQueryHook sets the given hook as the
// default binding hook.
func SetBindHeaderHook(bh BindHook) {
	if bh != nil {
		bindHeaderHook = bh
	}
}

// GetRenderHook returns the current render hook.
func GetRenderHook() RenderHook {
	return renderHook
}

// SetRenderHook sets the given hook as the default
// rendering hook. The media type is used to generate
// the OpenAPI specification.
func SetRenderHook(rh RenderHook) {
	if rh != nil {
		renderHook = rh
	}
}

// SetExecHook sets the given hook as the
// default execution hook.
func SetExecHook(eh ExecHook) {
	if eh != nil {
		execHook = eh
	}
}

// GetExecHook returns the current execution hook.
func GetExecHook() ExecHook {
	return execHook
}



// BindError is an error type returned when optic fails
// to bind parameters, to differentiate from errors returned
// by the handlers.
type BindError struct {
	validationErr error
	message       string
	typ           reflect.Type
	field         string
}

// Error implements the builtin error interface for BindError.
func (be BindError) Error() string {
	if be.field != "" && be.typ != nil {
		return fmt.Sprintf(
			"binding error on field '%s' of type '%s': %s",
			be.field,
			be.typ.Name(),
			be.message,
		)
	}
	return fmt.Sprintf("binding error: %s", be.message)
}

// ValidationErrors returns the errors from the validate process.
func (be BindError) ValidationErrors() validator.ValidationErrors {
	switch t := be.validationErr.(type) {
	case validator.ValidationErrors:
		return t
	}
	return nil
}

// An extractorFunc extracts data from a Fiber context according to
// parameters specified in a field tag.
type extractor func(*fiber.Ctx, string) (string, []string, error)

// extractQuery is an extractor tgat operated on the query
// parameters of a request.
func extractQuery(c *fiber.Ctx, tag string) (string, []string, error) {
	name, required, defaultVal, err := parseTagKey(tag)
	if err != nil {
		return "", nil, err
	}
	var params []string

	query := []string{c.Query(name)}

	if b, ok := c.Locals(ExplodeTag).(bool); ok && b {
		// Delete empty elements so default and required arguments
		// will play nice together. Append to a new collection to
		// preserve order without too much copying.
		params = make([]string, 0, len(query))
		for i := range query {
			if query[i] != "" {
				params = append(params, query[i])
			}
		}
	} else {
		splitFn := func(c rune) bool {
			return c == ','
		}
		if len(query) > 1 {
			return name, nil, errors.New("repeating values not supported: use comma-separated list")
		} else if len(query) == 1 {
			params = strings.FieldsFunc(query[0], splitFn)
		}
	}

	// XXX: deprecated, use of "default" tag is preferred
	if len(params) == 0 && defaultVal != "" {
		return name, []string{defaultVal}, nil
	}
	// XXX: deprecated, use of "validate" tag is preferred
	if len(params) == 0 && required {
		return "", nil, fmt.Errorf("missing query parameter: %s", name)
	}
	return name, params, nil
}

// extractPath is an extractor that operates on the path
// parameters of a request.
func extractPath(c *fiber.Ctx, tag string) (string, []string, error) {
	name, required, defaultVal, err := parseTagKey(tag)
	if err != nil {
		return "", nil, err
	}
	p := c.Params(name)

	// XXX: deprecated, use of "default" tag is preferred
	if p == "" && defaultVal != "" {
		return name, []string{defaultVal}, nil
	}
	// XXX: deprecated, use of "validate" tag is preferred
	if p == "" && required {
		return "", nil, fmt.Errorf("missing path parameter: %s", name)
	}

	return name, []string{p}, nil
}

// extractHeader is an extractor that operates on the headers
// of a request.
func extractHeader(c *fiber.Ctx, tag string) (string, []string, error) {
	name, required, defaultVal, err := parseTagKey(tag)
	if err != nil {
		return "", nil, err
	}
	header := c.Get(name)

	// XXX: deprecated, use of "default" tag is preferred
	if header == "" && defaultVal != "" {
		return name, []string{defaultVal}, nil
	}
	// XXX: deprecated, use of "validate" tag is preferred
	if required && header == "" {
		return "", nil, fmt.Errorf("missing header parameter: %s", name)
	}
	return name, []string{header}, nil
}

// Public signature does not expose "required" and "default" because
// they are deprecated in favor of the "validate" and "default" tags
func parseTagKey(tag string) (string, bool, string, error) {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return "", false, "", fmt.Errorf("empty tag")
	}
	name, options := parts[0], parts[1:]

	var defaultVal string

	// XXX: deprecated, required + default are kept here for backwards compatibility
	// use of "default" and "validate" tags is preferred
	// Iterate through the tag options to
	// find the required key.
	var required bool
	for _, o := range options {
		o = strings.TrimSpace(o)
		if o == RequiredTag {
			required = true
		} else if strings.HasPrefix(o, fmt.Sprintf("%s=", DefaultTag)) {
			defaultVal = strings.TrimPrefix(o, fmt.Sprintf("%s=", DefaultTag))
		} else {
			return "", false, "", fmt.Errorf("malformed tag for param '%s': unknown option '%s'", name, o)
		}
	}
	return name, required, defaultVal, nil
}

// ParseTagKey parses the given struct tag key and return the
// name of the field
func ParseTagKey(tag string) (string, error) {
	s, _, _, err := parseTagKey(tag)
	return s, err
}

// bindStringValue converts and bind the value s
// to the the reflected value v.
func bindStringValue(s string, v reflect.Value) error {
	// Ensure that the reflected value is addressable
	// and wasn't obtained by the use of an unexported
	// struct field, or calling a setter will panic.
	if !v.CanSet() {
		return fmt.Errorf("unaddressable value: %v", v)
	}
	i := reflect.New(v.Type()).Interface()

	// If the value implements the encoding.TextUnmarshaler
	// interface, bind the returned string representation.
	if unmarshaler, ok := i.(encoding.TextUnmarshaler); ok {
		if err := unmarshaler.UnmarshalText([]byte(s)); err != nil {
			return err
		}
		v.Set(reflect.Indirect(reflect.ValueOf(unmarshaler)))
		return nil
	}
	// Handle time.Duration.
	if _, ok := i.(time.Duration); ok {
		d, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(d))
	}
	// Switch over the kind of the reflected value
	// and convert the string to the proper type.
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(i)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.SetBool(b)
	case reflect.Float32, reflect.Float64:
		i, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(i)
	default:
		return fmt.Errorf("unsupported parameter type: %v", v.Kind())
	}
	return nil
}


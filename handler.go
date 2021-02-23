package optizz

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wI2L/fizz/openapi"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

type OptizzHandler struct {
	RouteInfo     *Route
	OperationInfo *openapi.OperationInfo
	Handler       fiber.Handler
}

// Optizz Handler is the wrapper of fiber.Handler with route and operation information.
func Handler(fiberHandler interface{}, status int, infos ...OperationOption) *OptizzHandler {
	hv := reflect.ValueOf(fiberHandler)

	if hv.Kind() != reflect.Func {
		panic(fmt.Sprintf("handler parameters must be a function, got %T", fiberHandler))
	}
	ht := hv.Type()
	fName := fmt.Sprintf("%s_%s", runtime.FuncForPC(hv.Pointer()).Name(), uuid.Must(uuid.NewRandom()).String())

	in := input(ht, fName)
	out := output(ht, fName)

	routeInfo := &Route{
		defaultStatusCode: status,
		handler:           hv,
		handlerType:       ht,
		inputType:         in,
		outputType:        out,
	}

	oi := &openapi.OperationInfo{}
	for _, info := range infos {
		info(oi)
	}

	f := func(c *fiber.Ctx) error {
		// funcIn contains the input parameters of the
		// optic handler call.
		args := []reflect.Value{reflect.ValueOf(c)}

		// Optic handler has custom input, handle
		// binding.
		if in != nil {
			input := reflect.New(in)
			// Bind the body with the hook.
			if err := bindHook(c, input); err != nil {
				handleError(c, BindError{message: err.Error(), typ: in})
				return err
			}
			// Bind query-parameters.
			if err := bindQueryHook(c, input); err != nil {
				handleError(c, err)
				return err
			}
			// Bind path arguments.
			if err := bindPathHook(c, input); err != nil {
				handleError(c, err)
				return err
			}
			// Bind headers.
			if err := bindHeaderHook(c, input); err != nil {
				handleError(c, err)
				return err
			}
			// validating query and path inputs if they have a validate tag
			initValidator()
			args = append(args, input)
			if err := validatorObj.Struct(input.Interface()); err != nil {
				handleError(c, BindError{message: err.Error(), validationErr: err})
				return err
			}
		}
		// Call optic handler with the arguments
		// and extract the returned values.
		var err, val interface{}

		ret := hv.Call(args)
		if out != nil {
			val = ret[0].Interface()
			err = ret[1].Interface()
		} else {
			err = ret[0].Interface()
		}
		// Handle the error returned by the
		// handler invocation, if any.
		if err != nil {
			handleError(c, err.(error))
			return err.(error)
		}
		renderHook(c, status, val)
		return nil
	}

	return &OptizzHandler{
		RouteInfo:     routeInfo,
		OperationInfo: oi,
		Handler:       f,
	}
}

// input checks the input parameters of a optic handler
// and return the type of the second parameter, if any.
func input(ht reflect.Type, name string) reflect.Type {
	n := ht.NumIn()
	if n < 1 || n > 2 {
		panic(fmt.Sprintf(
			"incorrect number of input parameters for handler %s, expected 1 or 2, got %d",
			name, n,
		))
	}
	// First parameter of optic handler must be
	// a pointer to a Fiber context.
	if !ht.In(0).ConvertibleTo(reflect.TypeOf(&fiber.Ctx{})) {
		panic(fmt.Sprintf(
			"invalid first parameter for handler %s, expected *Fiber.Ctx, got %v",
			name, ht.In(0),
		))
	}
	if n == 2 {
		// Check the type of the second parameter
		// of the handler. Must be a pointer to a struct.
		if ht.In(1).Kind() != reflect.Ptr || ht.In(1).Elem().Kind() != reflect.Struct {
			panic(fmt.Sprintf(
				"invalid second parameter for handler %s, expected pointer to struct, got %v",
				name, ht.In(1),
			))
		} else {
			return ht.In(1).Elem()
		}
	}
	return nil
}

// output checks the output parameters of a optic handler
// and return the type of the return type, if any.
func output(ht reflect.Type, name string) reflect.Type {
	n := ht.NumOut()

	if n < 1 || n > 2 {
		panic(fmt.Sprintf(
			"incorrect number of output parameters for handler %s, expected 1 or 2, got %d",
			name, n,
		))
	}
	// Check the type of the error parameter, which
	// should always come last.
	if !ht.Out(n - 1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic(fmt.Sprintf(
			"unsupported type for handler %s output parameter: expected error interface, got %v",
			name, ht.Out(n-1),
		))
	}
	if n == 2 {
		t := ht.Out(0)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		return t
	}
	return nil
}



// bind binds the fields the fields of the input object in with
// the values of the parameters extracted from the Fiber context.
// It reads tag to know what to extract using the extractor func.
func bind(c *fiber.Ctx, v reflect.Value, tag string, extract extractor) error {
	t := v.Type()

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		field := v.Field(i)

		// Handle embedded fields with a recursive call.
		// If the field is a pointer, but is nil, we
		// create a new value of the same type, or we
		// take the existing memory address.
		if ft.Anonymous {
			if field.Kind() == reflect.Ptr {
				if field.IsNil() {
					field.Set(reflect.New(field.Type().Elem()))
				}
			} else {
				if field.CanAddr() {
					field = field.Addr()
				}
			}
			err := bind(c, field, tag, extract)
			if err != nil {
				return err
			}
			continue
		}
		tagValue := ft.Tag.Get(tag)
		if tagValue == "" {
			continue
		}
		// Set-up context for extractors.
		// Query.
		c.Locals(ExplodeTag, true) // default
		if explodeVal, ok := ft.Tag.Lookup(ExplodeTag); ok {
			if explode, err := strconv.ParseBool(explodeVal); err == nil && !explode {
				c.Locals(ExplodeTag, false)
			}
		}
		_, fieldValues, err := extract(c, tagValue)
		if err != nil {
			return BindError{field: ft.Name, typ: t, message: err.Error()}
		}
		// Extract default value and use it in place
		// if no values were returned.
		def, ok := ft.Tag.Lookup(DefaultTag)
		if ok && len(fieldValues) == 0 {
			if b, ok := c.Locals(ExplodeTag).(bool); ok && b {
				fieldValues = append(fieldValues, strings.Split(def, ",")...)
			} else {
				fieldValues = append(fieldValues, def)
			}
		}
		if len(fieldValues) == 0 {
			continue
		}
		// If the field is a nil pointer to a concrete type,
		// create a new addressable value for this type.
		if field.Kind() == reflect.Ptr && field.IsNil() {
			f := reflect.New(field.Type().Elem())
			field.Set(f)
		}
		// Dereference pointer.
		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}
		kind := field.Kind()

		// Multiple values can only be filled to types
		// Slice and Array.
		if len(fieldValues) > 1 && (kind != reflect.Slice && kind != reflect.Array) {
			return BindError{field: ft.Name, typ: t, message: "multiple values not supported"}
		}
		// Ensure that the number of values to fill does
		// not exceed the length of a field of type Array.
		if kind == reflect.Array {
			if field.Len() != len(fieldValues) {
				return BindError{field: ft.Name, typ: t, message: fmt.Sprintf(
					"parameter expect %d values, got %d", field.Len(), len(fieldValues)),
				}
			}
		}
		if kind == reflect.Slice || kind == reflect.Array {
			// Create a new slice with an adequate
			// length to set all the values.
			if kind == reflect.Slice {
				field.Set(reflect.MakeSlice(field.Type(), 0, len(fieldValues)))
			}
			for i, val := range fieldValues {
				v := reflect.New(field.Type().Elem()).Elem()
				err = bindStringValue(val, v)
				if err != nil {
					return BindError{field: ft.Name, typ: t, message: err.Error()}
				}
				if kind == reflect.Slice {
					field.Set(reflect.Append(field, v))
				}
				if kind == reflect.Array {
					field.Index(i).Set(v)
				}
			}
			continue
		}
		// Handle enum values.
		enum := ft.Tag.Get(EnumTag)
		if enum != "" {
			enumValues := strings.Split(strings.TrimSpace(enum), ",")
			if len(enumValues) != 0 {
				if !contains(enumValues, fieldValues[0]) {
					return BindError{field: ft.Name, typ: t, message: fmt.Sprintf(
						"parameter has not an acceptable value, %s=%v", EnumTag, enumValues),
					}
				}
			}
		}
		// Fill string value into input field.
		err = bindStringValue(fieldValues[0], field)
		if err != nil {
			return BindError{field: ft.Name, typ: t, message: err.Error()}
		}
	}
	return nil
}

// handleError handles any error raised during the execution
// of the wrapping Fiber-handler.
func handleError(c *fiber.Ctx, err error) {
	var errors []error
	_errs := c.Locals("_errors_")
	if _errs == nil {
		errors = make([]error, 0)
	} else {
		if _errors, ok := _errs.([]error); ok {
			errors = _errors
		} else {
			errors = make([]error, 0)
		}
	}

	errors = append(errors, err)
	c.Locals("_errors_", errors)

	code, resp := errorHook(c, err)
	renderHook(c, code, resp)
}

// contains returns whether in contain s.
func contains(in []string, s string) bool {
	for _, v := range in {
		if v == s {
			return true
		}
	}
	return false
}

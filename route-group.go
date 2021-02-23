package optizz

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/wI2L/fizz/openapi"
	"path"
	"reflect"
	"runtime"
	"strings"
)

// RouterGroup is an abstraction of a Fiber router group.
type RouterGroup struct {
	app         *fiber.App
	group       fiber.Router
	gen         *openapi.Generator
	path        string
	Name        string
	Description string
}

// Group creates a new group of routes.
func (g *RouterGroup) Group(path, name, description string, handlers ...fiber.Handler) *RouterGroup {
	// Create the tag in the specification
	// for this groups.
	g.gen.AddTag(name, description)

	return &RouterGroup{
		app:         g.app,
		gen:         g.gen,
		group:       g.group.Group(path, handlers...),
		path:        joinPaths(g.path, path),
		Name:        name,
		Description: description,
	}
}

// Use adds middleware to the group.
func (g *RouterGroup) Use(handlers ...fiber.Handler) {
	for _, h := range handlers {
		g.group.Use(h)
	}
}

func (g *RouterGroup) Get(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle(path, "GET", handler, middlewares...)
}
func (g *RouterGroup) Post(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle(path, "POST", handler, middlewares...)
}
func (g *RouterGroup) Put(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle(path, "PUT", handler, middlewares...)
}
func (g *RouterGroup) Patch(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle(path, "PATCH", handler, middlewares...)
}
func (g *RouterGroup) Delete(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle(path, "DELETE", handler, middlewares...)
}
func (g *RouterGroup) Options(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle(path, "OPTIONS", handler, middlewares...)
}
func (g *RouterGroup) Head(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle(path, "HEAD", handler, middlewares...)
}
func (g *RouterGroup) Trace(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle(path, "TRACE", handler, middlewares...)
}

func (g *RouterGroup) Handle(path, method string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	handlers := make([]fiber.Handler, 0)
	if middlewares != nil && len(middlewares) > 0 {
		handlers = append(handlers, middlewares...)
	}

	// Optizz handler with OpenAPI style
	if handler != nil {
		ri := handler.RouteInfo
		oi := handler.OperationInfo
		
		// Set an operation ID if none is provided.
		if oi.ID == "" {
			// [method]-[full path]-[function name]
			abs := joinPaths(g.path, path)
			abs = strings.ReplaceAll(abs, "/", "-")
			oi.ID = fmt.Sprintf("%s-%s-%s", method, abs, ri.HandlerName())
		}
		oi.StatusCode = ri.GetDefaultStatusCode()

		// Set an input type if provided.
		it := ri.InputType()
		if oi.InputModel != nil {
			it = reflect.TypeOf(oi.InputModel)
		}

		// Consolidate path for OpenAPI spec.
		operationPath := joinPaths(g.path, path)
		// Add operation to the OpenAPI spec.
		_, err := g.gen.AddOperation(operationPath, method, g.Name, it, ri.OutputType(), oi)
		if err != nil {
			panic(fmt.Sprintf("error while generating OpenAPI spec on operation %s %s: %s", method, path, err))
		}

		handlers = append(handlers, handler.Handler)
	}

	g.group.Add(method, path, handlers...)
	return g
}

func joinPaths(abs, rel string) string {
	if rel == "" {
		return abs
	}
	final := path.Join(abs, rel)
	as := lastChar(rel) == '/' && lastChar(final) != '/'
	if as {
		return final + "/"
	}
	return final
}

func lastChar(str string) uint8 {
	if str == "" {
		panic("empty string")
	}
	return str[len(str)-1]
}

func funcEqual(f1, f2 interface{}) bool {
	v1 := reflect.ValueOf(f1)
	v2 := reflect.ValueOf(f2)

	if v1.Kind() == reflect.Func && v2.Kind() == reflect.Func { // prevent panic on call to Pointer()
		return runtime.FuncForPC(v1.Pointer()).Entry() == runtime.FuncForPC(v2.Pointer()).Entry()
	}
	return false
}
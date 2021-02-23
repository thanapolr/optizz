package optizz

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/wI2L/fizz/openapi"
	"path"
	"reflect"
	"runtime"
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
	return g.Handle2(path, "GET", handler, middlewares...)
}
func (g *RouterGroup) Post(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle2(path, "POST", handler, middlewares...)
}
func (g *RouterGroup) Put(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle2(path, "PUT", handler, middlewares...)
}
func (g *RouterGroup) Patch(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle2(path, "PATCH", handler, middlewares...)
}
func (g *RouterGroup) Delete(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle2(path, "DELETE", handler, middlewares...)
}
func (g *RouterGroup) Options(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle2(path, "OPTIONS", handler, middlewares...)
}
func (g *RouterGroup) Head(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle2(path, "HEAD", handler, middlewares...)
}
func (g *RouterGroup) Trace(path string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	return g.Handle2(path, "TRACE", handler, middlewares...)
}

func (g *RouterGroup) Handle2(path, method string, handler *OptizzHandler, middlewares ...fiber.Handler) *RouterGroup {
	ri := handler.RouteInfo
	oi := handler.OperationInfo

	// TODO better ID e.g. [method]-[full path]-[function name]
	// Set an operation ID if none is provided.
	if oi.ID == "" {
		oi.ID = ri.HandlerName()
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
		panic(fmt.Sprintf(
			"error while generating OpenAPI spec on operation %s %s: %s",
			method, path, err,
		))
	}

	handlers := make([]fiber.Handler, 0)
	if middlewares != nil && len(middlewares) > 0 {
		handlers = append(handlers, middlewares...)
	}
	handlers = append(handlers, handler.Handler)

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



//// GET is a shortcut to register a new handler with the GET method.
//func (g *RouterGroup) GET(path string, infos []OperationOption, handlers ...fiber.Handler) *RouterGroup {
//	return g.Handle(path, "GET", infos, handlers...)
//}
//
//// POST is a shortcut to register a new handler with the POST method.
//func (g *RouterGroup) POST(path string, infos []OperationOption, handlers ...fiber.Handler) *RouterGroup {
//	return g.Handle(path, "POST", infos, handlers...)
//}
//
//// PUT is a shortcut to register a new handler with the PUT method.
//func (g *RouterGroup) PUT(path string, infos []OperationOption, handlers ...fiber.Handler) *RouterGroup {
//	return g.Handle(path, "PUT", infos, handlers...)
//}
//
//// PATCH is a shortcut to register a new handler with the PATCH method.
//func (g *RouterGroup) PATCH(path string, infos []OperationOption, handlers ...fiber.Handler) *RouterGroup {
//	return g.Handle(path, "PATCH", infos, handlers...)
//}
//
//// DELETE is a shortcut to register a new handler with the DELETE method.
//func (g *RouterGroup) DELETE(path string, infos []OperationOption, handlers ...fiber.Handler) *RouterGroup {
//	return g.Handle(path, "DELETE", infos, handlers...)
//}
//
//// OPTIONS is a shortcut to register a new handler with the OPTIONS method.
//func (g *RouterGroup) OPTIONS(path string, infos []OperationOption, handlers ...fiber.Handler) *RouterGroup {
//	return g.Handle(path, "OPTIONS", infos, handlers...)
//}
//
//// HEAD is a shortcut to register a new handler with the HEAD method.
//func (g *RouterGroup) HEAD(path string, infos []OperationOption, handlers ...fiber.Handler) *RouterGroup {
//	return g.Handle(path, "HEAD", infos, handlers...)
//}
//
//// TRACE is a shortcut to register a new handler with the TRACE method.
//func (g *RouterGroup) TRACE(path string, infos []OperationOption, handlers ...fiber.Handler) *RouterGroup {
//	return g.Handle(path, "TRACE", infos, handlers...)
//}
//
//// Handle registers a new request handler that is wrapped
//// with Tonic and documented in the OpenAPI specification.
//func (g *RouterGroup) Handle(path, method string, infos []OperationOption, handlers ...fiber.Handler) *RouterGroup {
//	oi := &openapi.OperationInfo{}
//	for _, info := range infos {
//		info(oi)
//	}
//	type wrap struct {
//		h fiber.Handler
//		r *optic.Route
//	}
//	var wrapped []wrap
//
//	// Find the handlers wrapped with Tonic.
//	ctx := g.app.AcquireCtx(&fasthttp.RequestCtx{})
//	for _, h := range handlers {
//		r, err := optic.GetRouteByHandler(h, ctx)
//		if err == nil {
//			wrapped = append(wrapped, wrap{h: h, r: r})
//		}
//	}
//	g.app.ReleaseCtx(ctx)
//	// Check that no more that one tonic-wrapped handler
//	// is registered for this operation.
//	if len(wrapped) > 1 {
//		panic(fmt.Sprintf("multiple tonic-wrapped handler used for operation %s %s", method, path))
//	}
//	// If we have a tonic-wrapped handler, generate the
//	// specification of this operation.
//	if len(wrapped) == 1 {
//		hfunc := wrapped[0].r
//
//		// Set an operation ID if none is provided.
//		if oi.ID == "" {
//			oi.ID = hfunc.HandlerName()
//		}
//		oi.StatusCode = hfunc.GetDefaultStatusCode()
//
//		// Set an input type if provided.
//		it := hfunc.InputType()
//		if oi.InputModel != nil {
//			it = reflect.TypeOf(oi.InputModel)
//		}
//
//		// Consolidate path for OpenAPI spec.
//		operationPath := joinPaths(g.path, path)
//
//		// Add operation to the OpenAPI spec.
//		operation, err := g.gen.AddOperation(operationPath, method, g.Name, it, hfunc.OutputType(), oi)
//		if err != nil {
//			panic(fmt.Sprintf(
//				"error while generating OpenAPI spec on operation %s %s: %s",
//				method, path, err,
//			))
//		}
//		// If an operation was generated for the handler,
//		// wrap the Tonic-wrapped handled with a closure
//		// to inject it into the Fiber context.
//		if operation != nil {
//			for i, h := range handlers {
//				if funcEqual(h, wrapped[0].h) {
//					orig := h // copy the original func
//					handlers[i] = func(c *fiber.Ctx) error {
//						c.Locals(ctxOpenAPIOperation, operation)
//						return orig(c)
//					}
//				}
//			}
//		}
//	}
//	// Register the handlers with Fiber underlying group.
//	g.group.Add(method, path, handlers...)
//
//	return g
//}

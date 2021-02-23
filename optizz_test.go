package optizz

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"testing"
)

func BenchmarkFiber_App(b *testing.B) {
	app := fiber.New()
	for n := 0; n < b.N; n++ {
		app.Post(fmt.Sprint(n), func(c *fiber.Ctx) error { return c.JSON(n) })
	}
}

// BenchmarkOptizz_Handler-16    	  957356	      1114 ns/op
func BenchmarkOptizz_Handler(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Handler(func(c *fiber.Ctx) error { return c.JSON(n) }, 200)
	}
}

//BenchmarkRouteGroup_HandleNoOptizzHandler-16    	 1560498	       725 ns/op
func BenchmarkRouteGroup_HandleNoOptizzHandler(b *testing.B) {
	app := New()
	for n := 0; n < b.N; n++ {
		app.Post(fmt.Sprint(n), nil, func(c *fiber.Ctx) error { return nil })
	}
}

//BenchmarkRouteGroup_Handle-16    	  238382	      5814 ns/op
func BenchmarkRouteGroup_Handle(b *testing.B) {
	app := New()
	for n := 0; n < b.N; n++ {
		h := Handler(func(c *fiber.Ctx) error { return c.JSON(n) }, 200)
		app.Post(fmt.Sprint(n), h, func(c *fiber.Ctx) error { return nil })
	}
}

//BenchmarkOptizz_CallHandler-16    	 3440211	       360 ns/op
func BenchmarkOptizz_CallHandler(b *testing.B) {
	app := fiber.New()

	h := Handler(func(c *fiber.Ctx) error { return nil }, 200)
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	for n := 0; n < b.N; n++ {
		h.Handler(ctx)
	}
}

func BenchmarkFiber_CallHandler(b *testing.B) {
	app := fiber.New()

	h := func(c *fiber.Ctx) error { return nil }
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	for n := 0; n < b.N; n++ {
		h(ctx)
	}
}



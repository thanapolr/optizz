package main

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/thanapolr/optizz"
	"github.com/wI2L/fizz/openapi"
)

func main() {
	app := fiber.New()
	z := optizz.NewFromApp(app)
	api := z.Group("api", "api", "API routes")
	{
		api.Post("ping/:path1", optizz.Handler(pingPongHandler, 200,
			optizz.Summary("this is a summary"),
			optizz.Description("ping pong"),
		))
	}
	app.Get("openapi.json", z.OpenAPI(&openapi.Info{
		Title:       "example",
		Description: "example",
		Version:     "1.0.0",
	}, "json"))

	app.Get("hello", func(ctx *fiber.Ctx) error {
		return ctx.Format("hello")
	})

	app.Listen(":8080")
}

type Input struct {
	Header1    string                 `header:"X-Header-1"`
	Path1      string                 `path:"path1"`
	Query1     string                 `query:"query1"`
	BodyString string                 `json:"body_string" validate:"required"`
	BodyNumber json.Number            `json:"body_number"`
	BodyNested map[string]interface{} `json:"body_nested"`
}
type Output struct {
	Header1    string                 `json:"X-Header-1"`
	Path1      string                 `json:"path1"`
	Query1     string                 `json:"query1"`
	BodyString string                 `json:"body_string"`
	BodyNumber json.Number            `json:"body_number"`
	BodyNested map[string]interface{} `json:"body_nested"`
}

func pingPongHandler(c *fiber.Ctx, input *Input) (*Output, error) {
	out := &Output{
		Header1:    input.Header1,
		Path1:      input.Path1,
		Query1:     input.Query1,
		BodyString: input.BodyString,
		BodyNumber: input.BodyNumber,
		BodyNested: input.BodyNested,
	}
	return out, nil
}

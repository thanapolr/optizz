# Optizz
Optizz := [Tonic](https://github.com/loopfz/gadgeto) + [wI2L/Fizz](https://github.com/wI2L/fizz) 

* Tonic handle style for [Fiber](https://github.com/gofiber/fiber)
* wI2L/Fizz OpenAPI generator 


The original code are from Tonic and wI2L/Fizz and then modified to work with Fiber.

## Example
```
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
```

## OpenAPI
`curl localhost:8080/openapi.json`

```json
{
  "openapi": "3.0.1",
  "info": {
    "title": "example",
    "description": "example",
    "version": "1.0.0"
  },
  "paths": {
    "api/ping/{path1}": {
      "post": {
        "tags": [
          "api"
        ],
        "summary": "this is a summary",
        "description": "ping pong",
        "operationId": "pingPongHandler",
        "parameters": [
          {
            "name": "path1",
            "in": "path",
            "required": true,
            "schema": {
              "type": "string"
            }
          },
          {
            "name": "query1",
            "in": "query",
            "schema": {
              "type": "string"
            }
          },
          {
            "name": "X-Header-1",
            "in": "header",
            "schema": {
              "type": "string"
            }
          }
        ],
        "requestBody": {
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/PingPongHandlerInput"
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/Output"
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Output": {
        "type": "object",
        "properties": {
          "X-Header-1": {
            "type": "string"
          },
          "body_nested": {
            "type": "object",
            "additionalProperties": {}
          },
          "body_number": {
            "type": "string"
          },
          "body_string": {
            "type": "string"
          },
          "path1": {
            "type": "string"
          },
          "query1": {
            "type": "string"
          }
        }
      },
      "PingPongHandlerInput": {
        "type": "object",
        "properties": {
          "body_nested": {
            "type": "object",
            "additionalProperties": {}
          },
          "body_number": {
            "type": "string"
          },
          "body_string": {
            "type": "string"
          }
        },
        "required": [
          "body_string"
        ]
      }
    }
  },
  "tags": [
    {
      "name": "api",
      "description": "API routes"
    }
  ]
}
```
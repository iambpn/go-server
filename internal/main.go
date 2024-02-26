package main

import (
	"log"

	"github.com/iambpn/go-server/pkg/goServe"
	"github.com/iambpn/go-server/pkg/request"
	"github.com/iambpn/go-server/pkg/response"
)

func main() {
	app := goServe.New()

	app.AddPath("get", "/", func(req *request.Request, res *response.Response) error {
		err := res.JSON(response.JSONType{
			"ok": "ok get",
		})

		if err != nil {
			return err
		}

		res.Send()

		return nil
	})

	app.AddPath("post", "/", func(req *request.Request, res *response.Response) error {
		err := res.JSON(response.JSONType{
			"ok": "ok post",
		})

		if err != nil {
			return err
		}

		res.Send()
		return nil
	})

	err := app.Listen("127.0.0.1:8080")

	if err != nil {
		log.Fatalln(err)
	}
}

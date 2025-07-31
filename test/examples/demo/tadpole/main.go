package main

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/protocal/serialize/json"
	"github.com/lonng/nano/test/examples/demo/tadpole/logic"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()

	app.Name = "tadpole"
	app.Version = "0.0.1"
	app.Copyright = "nano authors reserved"
	app.Usage = "tadpole"

	// flags
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "addr",
			Value: ":23456",
			Usage: "game server address",
		},
	}

	app.Action = serve

	app.Run(os.Args)
}

func srcPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

func serve(ctx *cli.Context) error {
	components := &component.Components{}
	components.Register(logic.NewManager())
	components.Register(logic.NewWorld())

	// register all service
	options := []nano.Option{
		nano.WithComponents(components),
		nano.WithSerializer(json.NewSerializer()),
		nano.WithCheckOrigin(func(_ *http.Request) bool { return true }),
	}

	addr := ctx.String("addr")
	e := nano.New(options...)
	err := e.Startup()
	if err != nil {
		return err
	}

	log.Info("Open http://127.0.0.1:23456/static/ in browser")
	webDir := filepath.Join(srcPath(), "static")
	mux := http.NewServeMux()
	mux.Handle("/", e)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(webDir))))
	return http.ListenAndServe(addr, mux)
}

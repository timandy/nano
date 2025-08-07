package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/lonng/nano/cluster"
	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/protocal/serialize/json"
	"github.com/lonng/nano/test/examples/customerroute/onegate"
	"github.com/lonng/nano/test/examples/customerroute/tworoom"

	"github.com/lonng/nano"
	"github.com/lonng/nano/session"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "NanoCustomerRouteDemo"
	app.Description = "Nano cluster demo"
	app.Commands = []*cli.Command{
		{
			Name: "master",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "listen,l",
					Usage: "Master service listen address",
					Value: "127.0.0.1:34567",
				},
			},
			Action: runMaster,
		},
		{
			Name: "gate",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "master",
					Usage: "master server address",
					Value: "127.0.0.1:34567",
				},
				&cli.StringFlag{
					Name:  "listen,l",
					Usage: "Gate service listen address",
					Value: "",
				},
				&cli.StringFlag{
					Name:  "gate-address",
					Usage: "Client connect address",
					Value: "",
				},
			},
			Action: runGate,
		},
		{
			Name: "chat",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "master",
					Usage: "master server address",
					Value: "127.0.0.1:34567",
				},
				&cli.StringFlag{
					Name:  "listen,l",
					Usage: "Chat service listen address",
					Value: "",
				},
			},
			Action: runChat,
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal("Startup server error.", err)
	}
}

func srcPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

func runMaster(args *cli.Context) error {
	listen := args.String("listen")
	if listen == "" {
		return errors.New("master listen address cannot empty")
	}

	webDir := filepath.Join(srcPath(), "onemaster", "web")
	log.Info("Nano master server web content directory", webDir)
	log.Info("Nano master listen address", listen)
	log.Info("Open http://127.0.0.1:12345/web/ in browser")

	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir(webDir))))
	go func() {
		if err := http.ListenAndServe(":12345", nil); err != nil {
			panic(err)
		}
	}()

	// Startup Nano server with the specified listen address
	engine := nano.New(
		nano.WithNodeType(cluster.NodeTypeMaster),
		nano.WithServiceAddr(listen),
		nano.WithSerializer(json.NewSerializer()),
		nano.WithDebugMode(),
	)
	return engine.Run()
}

func runGate(args *cli.Context) error {
	listen := args.String("listen")
	if listen == "" {
		return errors.New("gate listen address cannot empty")
	}

	masterAddr := args.String("master")
	if masterAddr == "" {
		return errors.New("master address cannot empty")
	}

	gateAddr := args.String("gate-address")
	if gateAddr == "" {
		return errors.New("gate address cannot empty")
	}

	log.Info("Current server listen address", listen)
	log.Info("Current gate server address", gateAddr)
	log.Info("Remote master server address", masterAddr)

	// Startup Nano server with the specified listen address
	engine := nano.New(
		nano.WithNodeType(cluster.NodeTypeGate),
		nano.WithAdvertiseAddr(masterAddr),
		nano.WithServiceAddr(listen),
		nano.WithComponents(onegate.Services),
		nano.WithSerializer(json.NewSerializer()),
		nano.WithCheckOrigin(func(_ *http.Request) bool { return true }),
		nano.WithDebugMode(),
		nano.WithRemoteServiceRoute(customerRemoteServiceRoute), //set remote service route for gate
	)
	return engine.RunWs(gateAddr, "/nano")
}

func runChat(args *cli.Context) error {
	listen := args.String("listen")
	if listen == "" {
		return errors.New("chat listen address cannot empty")
	}

	masterAddr := args.String("master")
	if masterAddr == "" {
		return errors.New("master address cannot empty")
	}

	log.Info("Current chat server listen address", listen)
	log.Info("Remote master server address", masterAddr)

	// Register session closed callback
	session.Event.SessionClosed(tworoom.OnSessionClosed)

	// Startup Nano server with the specified listen address
	engine := nano.New(
		nano.WithNodeType(cluster.NodeTypeWorker),
		nano.WithAdvertiseAddr(masterAddr),
		nano.WithServiceAddr(listen),
		nano.WithComponents(tworoom.Services),
		nano.WithSerializer(json.NewSerializer()),
		nano.WithDebugMode(),
	)
	return engine.Run()
}

func customerRemoteServiceRoute(service string, session *session.Session, members []*clusterpb.MemberInfo) *clusterpb.MemberInfo {
	count := int64(len(members))
	var index = session.UID() % count
	fmt.Printf("remote service:%s route to :%v \n", service, members[index])
	return members[index]
}

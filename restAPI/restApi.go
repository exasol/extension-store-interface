package restAPI

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/exasol/extension-manager/extensionController"
	log "github.com/sirupsen/logrus"
)

// RestAPI is the interface that provides the REST API server of the extension-manager.
type RestAPI interface {
	// Serve starts the server. This method blocks until the server is stopped or fails.
	Serve()
	// Stop stops the server.
	Stop()
	// StartInBackground starts the server in the background and blocks until it is ready.
	StartInBackground()
}

// Create creates a new RestAPI.
func Create(controller extensionController.TransactionController, serverAddress string) RestAPI {
	return &restAPIImpl{controller: controller, serverAddress: serverAddress}
}

type restAPIImpl struct {
	controller    extensionController.TransactionController
	serverAddress string
	server        *http.Server
	stopped       *bool
	stoppedMutex  *sync.Mutex
	waitGroup     *sync.WaitGroup
}

func (api *restAPIImpl) Serve() {
	if api.server != nil {
		panic("server already running")
	}
	api.setStopped(false)

	handler, _, err := setupStandaloneAPI(api.controller)
	if err != nil {
		log.Fatalf("failed to setup api: %v", err)
	}
	api.server = &http.Server{
		Addr:    api.serverAddress,
		Handler: handler,
	}
	api.startServer()
}

func (api *restAPIImpl) startServer() {
	ln, err := net.Listen("tcp", api.serverAddress)
	if err != nil {
		log.Fatalf("failed to listen on address %s: %v", api.serverAddress, err)
	}
	log.Printf("Starting server on %s...\n", api.serverAddress)
	api.waitGroup.Done()
	err = api.server.Serve(ln) // blocking
	if err != nil && !api.isStopped() {
		log.Fatalf("failed to start server: %v", err)
	}
}

func (api *restAPIImpl) StartInBackground() {
	api.waitGroup = &sync.WaitGroup{}
	api.waitGroup.Add(1)
	go api.Serve()
	api.waitGroup.Wait()
}

func (api *restAPIImpl) setStopped(stopped bool) {
	if api.stopped == nil {
		stopped := false
		api.stopped = &stopped
		api.stoppedMutex = &sync.Mutex{}
	}
	api.stoppedMutex.Lock()
	defer api.stoppedMutex.Unlock()
	*api.stopped = stopped
}

func (api *restAPIImpl) isStopped() bool {
	api.stoppedMutex.Lock()
	defer api.stoppedMutex.Unlock()
	return *api.stopped
}

func (api *restAPIImpl) Stop() {
	if api.server == nil {
		panic("cant stop server since it's not running")
	}
	api.setStopped(true)
	err := api.server.Shutdown(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to shutdown rest API server. Cause: %v", err))
	}
	api.server = nil
}

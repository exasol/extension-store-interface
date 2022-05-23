package respApi

import (
	cont "backend/extensionController"
	"context"
	"database/sql"
	"fmt"
	"github.com/exasol/exasol-driver-go"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
)

type RestApi interface {
	Serve()
	Stop()
}

func Create(controller cont.ExtensionController) RestApi {
	return &restApiImpl{Controller: controller}
}

type restApiImpl struct {
	Controller   cont.ExtensionController
	server       *http.Server
	stopped      *bool
	stoppedMutex *sync.Mutex
}

func (restApi *restApiImpl) Serve() {
	if restApi.server != nil {
		panic("server already running")
	}
	restApi.setStopped(false)
	router := gin.Default()
	router.GET("/extensions", restApi.handleGetExtensions)
	router.GET("/installations", restApi.handleGetInstallations)
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}
	restApi.server = srv
	err := restApi.server.ListenAndServe() // blocking
	if err != nil && !restApi.isStopped() {
		panic(fmt.Sprintf("failed to start rest API server. Cause: %v", err.Error()))
	}
}

func (restApi *restApiImpl) setStopped(stopped bool) {
	if restApi.stopped == nil {
		stopped := false
		restApi.stopped = &stopped
		restApi.stoppedMutex = &sync.Mutex{}
	}
	restApi.stoppedMutex.Lock()
	defer restApi.stoppedMutex.Unlock()
	*restApi.stopped = stopped
}

func (restApi *restApiImpl) isStopped() bool {
	restApi.stoppedMutex.Lock()
	defer restApi.stoppedMutex.Unlock()
	return *restApi.stopped
}

func (restApi *restApiImpl) Stop() {
	if restApi.server == nil {
		panic("cant stop server since it's not running")
	}
	restApi.setStopped(true)
	err := restApi.server.Shutdown(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to shutdown rest API server. Cause: %v", err.Error()))
	}
	restApi.server = nil
}

func (restApi *restApiImpl) handleGetExtensions(c *gin.Context) {
	response, err := restApi.getExtensions(c)
	restApi.sendResponse(c, response, err)
}

func (restApi *restApiImpl) getExtensions(c *gin.Context) (*ExtensionsResponse, error) {
	dbConnection, err := restApi.openDbConnection(c)
	if err != nil {
		return nil, err
	}
	defer closeDbConnection(dbConnection)
	extensions, err := restApi.Controller.GetAllExtensions()
	if err != nil {
		return nil, err
	}
	var convertedExtensions []ExtensionsResponseExtension
	for _, extension := range extensions {
		convertedExtensions = append(convertedExtensions, ExtensionsResponseExtension{Name: extension.Name, Description: extension.Description, InstallableVersions: extension.InstallableVersions})
	}
	response := ExtensionsResponse{
		Extensions: convertedExtensions,
	}
	return &response, nil
}

type ExtensionsResponse struct {
	Extensions []ExtensionsResponseExtension `json:"extensions"`
}

type ExtensionsResponseExtension struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	InstallableVersions []string `json:"installableVersions"`
}

func (restApi *restApiImpl) handleGetInstallations(c *gin.Context) {
	response, err := restApi.getInstallations(c)
	restApi.sendResponse(c, response, err)
}

func (restApi *restApiImpl) getInstallations(c *gin.Context) (*InstallationsResponse, error) {
	dbConnection, err := restApi.openDbConnection(c)
	if err != nil {
		return nil, err
	}
	defer closeDbConnection(dbConnection)
	installations, err := restApi.Controller.GetAllInstallations(dbConnection)
	if err != nil {
		return nil, err
	}
	var convertedInstallations []InstallationsResponseInstallation
	for _, installation := range installations {
		convertedInstallations = append(convertedInstallations, InstallationsResponseInstallation{installation.Name})
	}
	response := InstallationsResponse{
		Installations: convertedInstallations,
	}
	return &response, nil
}

func (restApi *restApiImpl) sendResponse(c *gin.Context, response interface{}, err error) {
	if err != nil {
		c.String(500, "Internal error.")
		fmt.Println(err.Error())
		return
	}
	c.JSON(200, response)
}

func closeDbConnection(database *sql.DB) {
	err := database.Close()
	if err != nil {
		// Strange but not critical. So we just log it and go on.
		fmt.Printf("failed to close db connection. Cause %v", err.Error())
	}
}

func (restApi *restApiImpl) openDbConnection(c *gin.Context) (*sql.DB, error) {
	database, err := sql.Open("exasol", exasol.NewConfig(c.GetString("dbUser"), c.GetString("dbPass")).Port(c.GetInt("dbPort")).Host(c.GetString("dbHost")).String())
	if err != nil {
		return nil, fmt.Errorf("failed to open a database connection. Cause: %v", err.Error())
	}
	return database, nil
}

type InstallationsResponse struct {
	Installations []InstallationsResponseInstallation `json:"installations"`
}

type InstallationsResponseInstallation struct {
	Name string `json:"name"`
}
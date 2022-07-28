package restAPI

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	cont "github.com/exasol/extension-manager/extensionController"

	"github.com/exasol/exasol-driver-go"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// docs are generated by Swag CLI, you have to import it.
	_ "github.com/exasol/extension-manager/generatedApiDocs"
)

// RestAPI is the interface that provides the REST API server of the extension-manager.
type RestAPI interface {
	// Serve starts the server. This method blocks until the server is stopped or fails.
	Serve()
	// Stop stops the server
	Stop()
}

// @title           Exasol extension manager REST API
// @version         0.1.0
// @description     This is a REST API for managing extensions in an Exasol database.

// @contact.name   Exasol Integration team
// @contact.email  opensource@exasol.com

// @license.name  MIT
// @license.url   https://github.com/exasol/extension-manager/blob/main/LICENSE

// @BasePath  /
// @accept json
// @produce json

// Create creates a new RestAPI.
func Create(controller cont.ExtensionController, serverAddress string) RestAPI {
	return &restAPIImpl{controller: controller, serverAddress: serverAddress}
}

type restAPIImpl struct {
	controller    cont.ExtensionController
	serverAddress string
	server        *http.Server
	stopped       *bool
	stoppedMutex  *sync.Mutex
}

func (restApi *restAPIImpl) Serve() {
	if restApi.server != nil {
		panic("server already running")
	}
	restApi.setStopped(false)
	router := gin.Default()
	router.GET("/extensions", restApi.handleGetExtensions)
	router.GET("/installations", restApi.handleGetInstallations)
	router.PUT("/installations", restApi.handlePutInstallation)
	router.PUT("/instances", restApi.handlePutInstance)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	srv := &http.Server{
		Addr:    restApi.serverAddress,
		Handler: router,
	}
	restApi.server = srv
	log.Printf("Starting server on %s...\n", restApi.serverAddress)
	err := restApi.server.ListenAndServe() // blocking
	if err != nil && !restApi.isStopped() {
		panic(fmt.Sprintf("failed to start rest API server. Cause: %v", err))
	}
}

func (restApi *restAPIImpl) setStopped(stopped bool) {
	if restApi.stopped == nil {
		stopped := false
		restApi.stopped = &stopped
		restApi.stoppedMutex = &sync.Mutex{}
	}
	restApi.stoppedMutex.Lock()
	defer restApi.stoppedMutex.Unlock()
	*restApi.stopped = stopped
}

func (restApi *restAPIImpl) isStopped() bool {
	restApi.stoppedMutex.Lock()
	defer restApi.stoppedMutex.Unlock()
	return *restApi.stopped
}

func (restApi *restAPIImpl) Stop() {
	if restApi.server == nil {
		panic("cant stop server since it's not running")
	}
	restApi.setStopped(true)
	err := restApi.server.Shutdown(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to shutdown rest API server. Cause: %v", err))
	}
	restApi.server = nil
}

// @Summary      Get all extensions
// @Description  Get a list of all available extensions.
// @Id           getExtensions
// @Produce      json
// @Success      200 {object} ExtensionsResponse
// @Param        dbHost query string true "Hostname of the Exasol DB to manage"
// @Param        dbPort query int true "Port number of the Exasol DB to manage"
// @Param        dbUser query string true "Username of the Exasol DB to manage"
// @Param        dbPass query string true "Password of the Exasol DB to manage"
// @Failure      500 {object} string
// @Router       /extensions [get]
func (restApi *restAPIImpl) handleGetExtensions(c *gin.Context) {
	response, err := restApi.getExtensions(c)
	restApi.sendResponse(c, response, err)
}

func (restApi *restAPIImpl) getExtensions(c *gin.Context) (*ExtensionsResponse, error) {
	dbConnectionWithNoAutocommit, err := restApi.openDBConnection(c)
	if err != nil {
		return nil, err
	}
	defer closeDbConnection(dbConnectionWithNoAutocommit)
	extensions, err := restApi.controller.GetAllExtensions(dbConnectionWithNoAutocommit)
	if err != nil {
		return nil, err
	}
	convertedExtensions := make([]ExtensionsResponseExtension, 0, len(extensions))
	for _, extension := range extensions {
		ext := ExtensionsResponseExtension{Id: extension.Id, Name: extension.Name, Description: extension.Description, InstallableVersions: extension.InstallableVersions}
		convertedExtensions = append(convertedExtensions, ext)
	}
	response := ExtensionsResponse{
		Extensions: convertedExtensions,
	}
	return &response, nil
}

// @Description Response containing all available extensions
type ExtensionsResponse struct {
	Extensions []ExtensionsResponseExtension `json:"extensions"` // All available extensions.
}

// @Description Extension information
type ExtensionsResponseExtension struct {
	Id                  string   `json:"id"`                  // ID of the extension. Don't store this as it may change in the future.
	Name                string   `json:"name"`                // The name of the extension to be displayed to the user.
	Description         string   `json:"description"`         // The description of the extension to be displayed to the user.
	InstallableVersions []string `json:"installableVersions"` // A list of versions of this extension available for installation.
}

// @Summary      Get all installations.
// @Description  Get a list of all installations. Installation means, that an extension is installed in the database (e.g. JAR files added to BucketFS, Adapter Script created).
// @Id           getInstallations
// @Produce      json
// @Success      200 {object} InstallationsResponse
// @Param        dbHost query string true "Hostname of the Exasol DB to manage"
// @Param        dbPort query int true "Port number of the Exasol DB to manage"
// @Param        dbUser query string true "Username of the Exasol DB to manage"
// @Param        dbPass query string true "Password of the Exasol DB to manage"
// @Failure      500 {object} string
// @Router       /installations [get]
func (restApi *restAPIImpl) handleGetInstallations(c *gin.Context) {
	response, err := restApi.getInstallations(c)
	restApi.sendResponse(c, response, err)
}

func (restApi *restAPIImpl) getInstallations(c *gin.Context) (*InstallationsResponse, error) {
	dbConnection, err := restApi.openDBConnection(c)
	if err != nil {
		return nil, err
	}
	defer closeDbConnection(dbConnection)
	installations, err := restApi.controller.GetAllInstallations(dbConnection)
	if err != nil {
		return nil, err
	}
	convertedInstallations := make([]InstallationsResponseInstallation, 0, len(installations))
	for _, installation := range installations {
		convertedInstallations = append(convertedInstallations, InstallationsResponseInstallation{installation.Name, installation.Version, installation.InstanceParameters})
	}
	response := InstallationsResponse{
		Installations: convertedInstallations,
	}
	return &response, nil
}

// @Summary      Install an extension.
// @Description  This installs an extension in a given version.
// @Id           installExtension
// @Produce      json
// @Success      200 {object} string
// @Param        dbHost query string true "Hostname of the Exasol DB to manage"
// @Param        dbPort query int true "Port number of the Exasol DB to manage"
// @Param        dbUser query string true "Username of the Exasol DB to manage"
// @Param        dbPass query string true "Password of the Exasol DB to manage"
// @Param        extensionId query string true "ID of the extension to install"
// @Param        extensionVersion query string true "Version of the extension to install"
// @Param        dummy body string false "dummy body" default()
// @Failure      500 {object} string
// @Router       /installations [put]
func (restApi *restAPIImpl) handlePutInstallation(c *gin.Context) {
	result, err := restApi.installExtension(c)
	restApi.sendResponse(c, result, err)
}

func (restApi *restAPIImpl) installExtension(c *gin.Context) (string, error) {
	dbConnection, err := restApi.openDBConnection(c)
	if err != nil {
		return "", err
	}
	defer closeDbConnection(dbConnection)
	query := c.Request.URL.Query()
	extensionId := query.Get("extensionId")
	if extensionId == "" {
		return "", fmt.Errorf("missing parameter extensionId")
	}
	extensionVersion := query.Get("extensionVersion")
	if extensionVersion == "" {
		return "", fmt.Errorf("missing parameter extensionVersion")
	}

	err = restApi.controller.InstallExtension(dbConnection, extensionId, extensionVersion)

	if err != nil {
		return "", fmt.Errorf("error installing extension: %v", err)
	}
	return "", nil
}

// @Summary      Create an instance of an extension.
// @Description  This creates an instance of an extension, e.g. a virtual schema.
// @Id           createInstance
// @Produce      json
// @Success      200 {object} string
// @Param        dbHost query string true "Hostname of the Exasol DB to manage"
// @Param        dbPort query int true "Port number of the Exasol DB to manage"
// @Param        dbUser query string true "Username of the Exasol DB to manage"
// @Param        dbPass query string true "Password of the Exasol DB to manage"
// @Param        createInstanceRequest body CreateInstanceRequest true "Request data for creating an instance"
// @Failure      500 {object} string
// @Router       /installations [put]
func (restApi *restAPIImpl) handlePutInstance(c *gin.Context) {
	result, err := restApi.createInstance(c)
	restApi.sendResponse(c, result, err)
}

func (restApi *restAPIImpl) createInstance(c *gin.Context) (string, error) {
	dbConnection, err := restApi.openDBConnection(c)
	if err != nil {
		return "", err
	}
	defer closeDbConnection(dbConnection)
	var request CreateInstanceRequest
	if err := c.BindJSON(&request); err != nil {
		return "", fmt.Errorf("invalid request: %w", err)
	}

	var parameters []cont.ParameterValue
	for _, p := range request.ParameterValues {
		parameters = append(parameters, cont.ParameterValue{Name: p.Name, Value: p.Value})
	}
	err = restApi.controller.CreateInstance(dbConnection, request.ExtensionId, request.ExtensionVersion, parameters)
	if err != nil {
		return "", fmt.Errorf("error installing extension: %v", err)
	}
	return "", nil
}

// @Description Request data for creating a new instance of an extension.
type CreateInstanceRequest struct {
	ExtensionId      string           `json:"extensionId"`      // The ID of the extension
	ExtensionVersion string           `json:"extensionVersion"` // The version of the extension
	ParameterValues  []ParameterValue `json:"parameterValues"`  // The parameters for the new instance
}

// @Description Parameter values for creating a new instance.
type ParameterValue struct {
	Name  string `json:"name"`  // The name of the parameter
	Value string `json:"value"` // The value of the parameter
}

func (restApi *restAPIImpl) sendResponse(c *gin.Context, response interface{}, err error) {
	if err != nil {
		c.String(500, "Internal error.")
		log.Printf("request failed: %v\n", err)
		return
	}
	if s, ok := response.(string); ok {
		c.String(200, s)
	} else {
		c.JSON(200, response)
	}
}

func closeDbConnection(database *sql.DB) {
	err := database.Close()
	if err != nil {
		// Strange but not critical. So we just log it and go on.
		fmt.Printf("failed to close db connection. Cause %v", err)
	}
}

func (restApi *restAPIImpl) openDBConnection(c *gin.Context) (*sql.DB, error) {
	config, err := getDbConfig(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get db config: %w", err)
	}
	config.Autocommit(false).ValidateServerCertificate(false)
	database, err := sql.Open("exasol", config.String())
	if err != nil {
		return nil, fmt.Errorf("failed to open a database connection. Cause: %w", err)
	}
	return database, nil
}

func getDbConfig(c *gin.Context) (*exasol.DSNConfigBuilder, error) {
	query := c.Request.URL.Query()
	host := query.Get("dbHost")
	if host == "" {
		return nil, fmt.Errorf("missing parameter dbHost")
	}
	portString := query.Get("dbPort")
	if portString == "" {
		return nil, fmt.Errorf("missing parameter dbPort")
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		return nil, fmt.Errorf("invalid value %q for parameter dbPort", portString)
	}
	user := query.Get("dbUser")
	if user == "" {
		return nil, fmt.Errorf("missing parameter dbUser")
	}
	password := query.Get("dbPass")
	if password == "" {
		return nil, fmt.Errorf("missing parameter dbPass")
	}
	config := exasol.NewConfig(user, password).Port(port).Host(host)
	return config, nil
}

type InstallationsResponse struct {
	Installations []InstallationsResponseInstallation `json:"installations"`
}

type InstallationsResponseInstallation struct {
	Name               string        `json:"name"`
	Version            string        `json:"version"`
	InstanceParameters []interface{} `json:"instanceParameters"`
}

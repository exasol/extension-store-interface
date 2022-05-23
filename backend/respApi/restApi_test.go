package respApi

import (
	"backend/extensionApi"
	"database/sql"
	"github.com/kinbiko/jsonassert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

type RestAPISuite struct {
	suite.Suite
	assertJSON *jsonassert.Asserter
	controller *MockExtensionController
	restAPI    RestApi
}

func TestRestApiSuite(t *testing.T) {
	suite.Run(t, new(RestAPISuite))
}

type MockExtensionController struct {
	mock.Mock
}

func (suite *RestAPISuite) SetupSuite() {
	suite.assertJSON = jsonassert.New(suite.T())
}

func (suite *RestAPISuite) SetupTest() {
	suite.controller = &MockExtensionController{}
	suite.restAPI = Create(suite.controller)
	go suite.restAPI.Serve()
	time.Sleep(10 * time.Millisecond) // give the server some time to become ready
}

func (suite *RestAPISuite) TearDownTest() {
	suite.restAPI.Stop()
}

func (mock *MockExtensionController) GetAllInstallations(dbConnection *sql.DB) ([]*extensionApi.Installation, error) {
	args := mock.Called(dbConnection)
	return args.Get(0).([]*extensionApi.Installation), args.Error(1)
}

func (mock *MockExtensionController) GetAllExtensions() ([]*extensionApi.Extension, error) {
	args := mock.Called()
	return args.Get(0).([]*extensionApi.Extension), args.Error(1)
}

func (suite *RestAPISuite) TestGetInstallations() {
	suite.controller.On("GetAllInstallations", mock.Anything).Return([]*extensionApi.Installation{{Name: "test"}}, nil)
	responseString := suite.makeGetRequest("/installations")
	suite.assertJSON.Assertf(responseString, `{"installations":[{"name":"test"}]}`)
}

func (suite *RestAPISuite) TestGetExtensions() {
	suite.controller.On("GetAllExtensions", mock.Anything).Return([]*extensionApi.Extension{{Name: "my-extension", Description: "a cool extension", InstallableVersions: []string{"0.1.0"}}}, nil)
	responseString := suite.makeGetRequest("/extensions")
	suite.assertJSON.Assertf(responseString, `{"extensions":[{"name":"my-extension","description":"a cool extension","installableVersions":["0.1.0"]}]}`)
}

func (suite *RestAPISuite) makeGetRequest(path string) string {
	const apiHost = "http://localhost:8080"
	response, err := http.Get(apiHost + path)
	suite.NoError(err)
	defer func() { suite.NoError(response.Body.Close()) }()
	bytes, err := ioutil.ReadAll(response.Body)
	suite.NoError(err)
	return string(bytes)
}
package restAPI

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/exasol/extension-manager/extensionController"
	"github.com/exasol/extension-manager/integrationTesting"
	"github.com/kinbiko/jsonassert"
	"github.com/stretchr/testify/suite"
)

const (
	EXTENSION_SCHEMA     = "test"
	DEFAULT_EXTENSION_ID = "testing-extension.js"
)

type RestAPIIntegrationTestSuite struct {
	integrationTesting.IntegrationTestSuite
	tempExtensionRepo string
	assertJSON        *jsonassert.Asserter
	restAPI           RestAPI
	baseUrl           string
}

func TestRestAPIIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RestAPIIntegrationTestSuite))
}

func (suite *RestAPIIntegrationTestSuite) SetupSuite() {
	suite.IntegrationTestSuite.SetupSuite()
	suite.assertJSON = jsonassert.New(suite.T())
}

func (suite *RestAPIIntegrationTestSuite) SetupTest() {
	ctrl := extensionController.Create(suite.tempExtensionRepo, EXTENSION_SCHEMA)
	suite.restAPI = Create(ctrl, "localhost:8081")
	suite.baseUrl = "http://localhost:8081"
	go suite.restAPI.Serve()
	time.Sleep(10 * time.Millisecond) // give the server some time to become ready
}

func (suite *RestAPIIntegrationTestSuite) TearDownTest() {
	suite.restAPI.Stop()
}

func (suite *RestAPIIntegrationTestSuite) TestGetAllExtensionsSuccessfully() {
	response := suite.makeGetRequest("/extensions?" + suite.getValidDbArgs())
	suite.assertJSON.Assertf(response, `{"extensions":[]}`)
}

func (suite *RestAPIIntegrationTestSuite) TestGetInstallationsSuccessfully() {
	response := suite.makeGetRequest("/installations?" + suite.getValidDbArgs())
	suite.assertJSON.Assertf(response, `{"installations":[]}`)
}

func (suite *RestAPIIntegrationTestSuite) TestGetInstallationsFails_InvalidCredentials() {
	var tests = []struct{ parameters string }{
		{parameters: suite.getDbArgsWithUserPassword("invalidUser", "password")},
		{parameters: suite.getDbArgsWithAccessToken("invalidAccessToken")},
		{parameters: suite.getDbArgsWithRefreshToken("invalidRefreshToken")}}
	for _, test := range tests {
		suite.Run(test.parameters, func() {
			response := suite.makeRequest("GET", "/installations?"+test.parameters, "", 500)
			suite.Regexp("Request failed: E-EGOD-11: execution failed with SQL error code '08004' and message 'Connection exception - authentication failed.*", response)
		})
	}
}

func (suite *RestAPIIntegrationTestSuite) getValidDbArgs() string {
	info := suite.ConnectionInfo
	return suite.getDbArgsWithUserPassword(info.User, info.Password)
}

func (suite *RestAPIIntegrationTestSuite) getDbArgsWithUserPassword(user string, password string) string {
	info := suite.ConnectionInfo
	return fmt.Sprintf("dbHost=%s&dbPort=%d&dbUser=%s&dbPass=%s", info.Host, info.Port, user, password)
}

func (suite *RestAPIIntegrationTestSuite) getDbArgsWithAccessToken(accessToken string) string {
	info := suite.ConnectionInfo
	return fmt.Sprintf("dbHost=%s&dbPort=%d&accessToken=%s", info.Host, info.Port, accessToken)
}

func (suite *RestAPIIntegrationTestSuite) getDbArgsWithRefreshToken(refreshToken string) string {
	info := suite.ConnectionInfo
	return fmt.Sprintf("dbHost=%s&dbPort=%d&refreshToken=%s", info.Host, info.Port, refreshToken)
}

func (suite *RestAPIIntegrationTestSuite) makeGetRequest(path string) string {
	return suite.makeRequest("GET", path, "", 200)
}

func (suite *RestAPIIntegrationTestSuite) makeRequest(method string, path string, body string, expectedStatusCode int) string {
	request, err := http.NewRequest(method, suite.baseUrl+path, strings.NewReader(body))
	suite.NoError(err)
	response, err := http.DefaultClient.Do(request)
	suite.NoError(err)
	suite.Equal(expectedStatusCode, response.StatusCode)
	defer func() { suite.NoError(response.Body.Close()) }()
	bytes, err := io.ReadAll(response.Body)
	suite.NoError(err)
	return string(bytes)
}
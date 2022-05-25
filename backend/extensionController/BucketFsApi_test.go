package extensionController

import (
	"backend/integrationTesting"
	"github.com/stretchr/testify/suite"
	"testing"
)

type BucketFsAPISuite struct {
	integrationTesting.IntegrationTestSuite
}

func TestBucketFsApiSuite(t *testing.T) {
	suite.Run(t, new(BucketFsAPISuite))
}

func (suite *BucketFsAPISuite) TestListBuckets() {
	connectionWithNoAutocommit := suite.Exasol.CreateConnectionWithConfig(false)
	defer func() { suite.NoError(connectionWithNoAutocommit.Close()) }()
	bfsAPI := CreateBucketFsAPI(connectionWithNoAutocommit)
	result, err := bfsAPI.ListBuckets()
	suite.NoError(err)
	suite.Assert().Contains(result, "default")
}

func (suite *BucketFsAPISuite) TestListFiles() {
	connectionWithNoAutocommit := suite.Exasol.CreateConnectionWithConfig(false)
	defer func() { suite.NoError(connectionWithNoAutocommit.Close()) }()
	bfsAPI := CreateBucketFsAPI(connectionWithNoAutocommit)
	suite.Exasol.UploadStringContent("12345", "myFile.txt")
	defer suite.Exasol.DeleteFile("myFile.txt")
	result, err := bfsAPI.ListFiles("default")
	suite.NoError(err)
	suite.Assert().Contains(result, BfsFile{Name: "myFile.txt", Size: 5})
}

# Developer Guide

This guide describes how to develop, test and build Extension Manager (EM).

If you want to embed EM into another application, see the [embedding guide](./embedding_extension_manager.md). If you want to develop an extension for EM, see the [extension developer guide](./extension_developer_guide.md).

## Extension Manager

This project uses a mix of different languages and build tools / helpers, so please read the instructions carefully.

### Project Keeper and Release Droid

For Java and Go dependency management and documentation this project uses [Project Keeper (PK)](github.com/exasol/project-keeper). On the other hand it also uses [Release Droid (RD)](github.com/exasol/release-droid) to create the release. RD is deprecated and will be replaced once PK can handle the complex project setup used in this project.

#### Updating the Dependency Documentation and Change Log

In this project you **cannot** run project keeper directly from the main directory like you are used to. Instead, please run the shell script that is also used in the GitHub action: 

To verify the current project:

```sh
.github/workflows/project-keeper.sh verify
```

To automatically fix it:

```sh
.github/workflows/project-keeper.sh fix
```

### Building

To build the binary, run

```sh
go generate ./...
go build -o extension-manager cmd/main.go
```

To run the extension manager, execute

```sh
# Show supported command line arguments
go run cmd/main.go -h
# Start server with custom extension registry
go run cmd/main.go -serverAddress localhost:8080 -extensionRegistryURL /path/to/extensions/
```

After starting the server you can get the OpenApi definition by executing

```sh
curl "http://localhost:8080/openapi.json" -o extension-manager-api.json
```

### Requirement Tracing

You can run requirements tracing by executing:

```sh
./ci/trace-requirements.sh
```

If tracing fails with a `org.xml.sax.SAXParseException` you might need to run `mvn clean` before to delete temporary files like `target/site/jacoco/jacoco.xml`.

### Testing

The extension-manager project contains unit and integration tests that verify
* Loading and executing of JavaScript extensions
* Database interactions
* REST API interface
* Server-side parameter validation using `extension-parameter-validator`

Tests use dummy extensions, no real extensions.

#### Non-Parallel Tests

The tests of this project use [`exasol-test-setup-abstraction-server`](https://github.com/exasol/exasol-test-setup-abstraction-server/). There the tests connect to an Exasol database running in a docker container. For performance reasons the test-setup-abstraction reuses that container. This feature is not compatible with running tests in parallel.

Problems would be:

* Name conflicts, e.g. schema names
* Missing isolation, e.g. `EXA_ALL_SCRIPTS` contains objects from other tests
* Issues with the exasol-test-setup-abstraction-server (the download of the server jar is triggered by the first test. The second one tries to use the unfinished jar)

For that reason parallel tests are currently disabled in the CI with `-p 1`.

To run test locally use:

```sh
go test -p 1 ./...
```

To run only tests without the database use:

```sh
go test -p 1 -short ./...
```

Please note that also `-short` tests need `-p 1` because extension integration tests share a directory for building a test extension. Tests will fail randomly without `-p 1`.

#### Manual Integration Tests With SaaS

Normal integration test EM against an Exasol DB running in a Docker container. There might be differences to a real Exasol SaaS DB. Tests against SaaS are not automated, you need to run them manually:

1. Create a Personal Access Token in SaaS with scope `databases:use`
2. Create a new schema you want to use for testing
3. Upload required Adapter JARs to BucketFS
4. Create file `manual-test.properties` with the following content:
   ```properties
   databaseHost = <SaaS DB host>
   databaseToken = <SaaS DB personal access token>
   extensionRegistryURL= <Registry URL>
   extensionSchema = <Extension Schema>
   bucketFSBasePath = <Bucket FS base path>
   ```
5. Run test with
   ```sh
   go test -v ./cmd/...
   ```

### Static Code Analysis

#### Go Linter

To install golangci-lint on your machine, follow [these instruction](https://golangci-lint.run/usage/install/#local-installation).

To run the linter, execute

```sh
golangci-lint run
```

File `.golangci.yml` contains configuration like enabled or disabled linters.

#### Sonar

Download sonar-scanner as a zip file from [sonarqube.org](https://docs.sonarqube.org/latest/analysis/scan/sonarscanner/) and unpack it.

Run tests to generate code coverage information:

```sh
go test -v -p 1 -count 1 -coverprofile=coverage.out ./...
mvn verify
```

Then run Sonar with the following command in the project root:

```sh
sonar-scanner -Dsonar.token=$SONAR_TOKEN
```

### Using a Local Extension Interface

To use a local, non-published version of the extension interface for testing EM follow these steps:

1. Build `extension-manager-interface` by running `npm run build`. This is required after each change.
2. Edit [pkg/integrationTesting/extensionForTesting/package.json](./../pkg/integrationTesting/extensionForTesting/package.json) and replace the version of `"@exasol/extension-manager-interface"` with the path to your local clone of [extension-manager-interface](https://github.com/exasol/extension-manager-interface).
3. Edit [pkg/integrationTesting/extensionForTesting/extensionForTestingTemplate.js](./../pkg/integrationTesting/extensionForTesting/extensionForTestingTemplate.js) and adapt it to the new API if necessary.

   **Note:** The file contains placeholders that will be replaced during tests. It is not valid JavaScript, so it's normal that the editor complains about the invalid syntax.

Make sure to not commit the modified `package.json`.

## Extension Registry

The extension registry is an HTTPS service that provides a JSON file containing links to all available extensions. The service consists of an S3 Bucket and a CloudFront distribution deployed via AWS Cloud Development Kit (CDK).

### Initial Configuration

1. Create file `registry/lib/config.ts` with the following content:
    ```ts
    export const CONFIG = {
        owner: 'your.email@example.com'
    }
    ```
2. Run `npm install`
3. Configure AWS profile and region:
    ```sh
    export AWS_PROFILE=<profile>
    export AWS_REGION=eu-central-1
    ```

### Deploy Changes

Run `npm run cdk diff`. If the output looks good, run `npm run cdk deploy`.

To get the output variables of the deployed stack (e.g. bucket name and CloudFront distribution host name), run the following command:

```sh
aws cloudformation describe-stacks --stack-name ExtensionManagerRegistry --query "Stacks[0].Outputs[].{key:ExportName,value:OutputValue}"
```

### Deploy Registry Content

#### Generate Registry Content

Run the following command to generate the registry content using the latest extension versions:

```sh
cd registry-upload
npm run generate
```

#### Upload Registry

To deploy the content of the Extension Registry to `test` or `prod` stage, run the following command:

```sh
cd registry-upload
AWS_PROFILE=$profile npm run upload -- --stage=test --no-dry-run
# or
AWS_PROFILE=$profile npm run upload -- --stage=prod --no-dry-run
```

This will upload the JSON file from the `registry-upload/content` folder for the given stage to the S3 bucket and invalidate the CloudFront cache. It will also upload the `testing-extension.js` extension to the `test` stage.

### Upgrade NPM Dependencies

```sh
npx npm-check-updates -u && npm install
```

## Python UDF

EM uses a Python UDF located at [pkg/extensionController/bfs/udf](../pkg/extensionController/bfs/udf) to list files in BucketFS.

### Initial Setup

```sh
cd pkg/extensionController/bfs/udf
poetry env use 3.8
poetry install
```

### Run tests

```sh
cd pkg/extensionController/bfs/udf
poetry run pytest
# Test coverage
poetry run pytest --cov=list_files_udf
```

### Run Type Check

```sh
cd pkg/extensionController/bfs/udf
poetry run mypy .
```

### Run Linter

```sh
cd pkg/extensionController/bfs/udf
poetry run pylint *.py
```

package requests

import (
	"database/sql"
	"net/http"

	"github.com/Nightapes/go-rest/pkg/openapi"

	"github.com/exasol/extension-manager/restAPI/core"
	"github.com/exasol/extension-manager/restAPI/requests/dbRequest"
)

func InstallExtension(apiContext *core.ApiContext) *openapi.Put {
	return &openapi.Put{
		Summary:        "Install an extension.",
		Description:    "This installs an extension in a given version, e.g. by creating Adapter Scripts.",
		OperationID:    "InstallExtension",
		Tags:           []string{core.TagExtension},
		Authentication: authentication,
		RequestBody:    InstallExtensionRequest{},
		Response: map[string]openapi.MethodResponse{
			"204": {Description: "OK"},
		},
		Path:        newPathWithDbQueryParams().Add("installations"),
		HandlerFunc: dbRequest.CreateHandler(handleInstallExtension(apiContext)),
	}
}

func handleInstallExtension(apiContext *core.ApiContext) dbRequest.DbHandler {
	return func(db *sql.DB, writer http.ResponseWriter, request *http.Request) {
		requestBody := InstallExtensionRequest{}
		err := core.DecodeJSONBody(writer, request, &requestBody)
		if err != nil {
			core.HandleError(request.Context(), writer, err)
			return
		}
		err = apiContext.Controller.InstallExtension(request.Context(), db, requestBody.ExtensionId, requestBody.ExtensionVersion)

		if err != nil {
			core.HandleError(request.Context(), writer, err)
			return
		}
		core.SendNoContent(request.Context(), writer)
	}
}

type InstallExtensionRequest struct {
	ExtensionId      string
	ExtensionVersion string
}

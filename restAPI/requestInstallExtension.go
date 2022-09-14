package restAPI

import (
	"database/sql"
	"net/http"

	"github.com/Nightapes/go-rest/pkg/openapi"
	"github.com/go-chi/chi/v5"
)

func InstallExtension(apiContext *ApiContext) *openapi.Put {
	return &openapi.Put{
		Summary:        "Install an extension.",
		Description:    "This installs an extension in a given version, e.g. by creating Adapter Scripts.",
		OperationID:    "InstallExtension",
		Tags:           []string{TagExtension},
		Authentication: authentication,
		Response: map[string]openapi.MethodResponse{
			"204": {Description: "OK"},
		},
		Path: newPathWithDbQueryParams().
			Add("extensions").
			AddParameter("extensionId", openapi.STRING, "ID of the extension to install").
			AddParameter("extensionVersion", openapi.STRING, "Version of the extension to install").
			Add("install"),
		HandlerFunc: adaptDbHandler(handleInstallExtension(apiContext)),
	}
}

func handleInstallExtension(apiContext *ApiContext) dbHandler {
	return func(db *sql.DB, writer http.ResponseWriter, request *http.Request) {
		extensionId := chi.URLParam(request, "extensionId")
		extensionVersion := chi.URLParam(request, "extensionVersion")
		err := apiContext.Controller.InstallExtension(request.Context(), db, extensionId, extensionVersion)

		if err != nil {
			HandleError(request.Context(), writer, err)
			return
		}
		SendNoContent(request.Context(), writer)
	}
}

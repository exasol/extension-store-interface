package restAPI

import (
	"database/sql"
	"net/http"

	"github.com/Nightapes/go-rest/pkg/openapi"
	"github.com/go-chi/chi/v5"
)

func UninstallExtension(apiContext *ApiContext) *openapi.Delete {
	return &openapi.Delete{
		Summary:        "Uninstall an extension.",
		Description:    "This uninstalls an extension in a given version, e.g. by removing Adapter Scripts.",
		OperationID:    "UninstallExtension",
		Tags:           []string{TagInstallation},
		Authentication: authentication,
		Response: map[string]openapi.MethodResponse{
			"204": {Description: "OK"},
		},
		Path: newPathWithDbQueryParams().
			Add("installations").
			AddParameter("extensionId", openapi.STRING, "The ID of the extension to uninstall").
			AddParameter("extensionVersion", openapi.STRING, "The version of the extension to uninstall"),
		HandlerFunc: adaptDbHandler(handleUninstallExtension(apiContext)),
	}
}

func handleUninstallExtension(apiContext *ApiContext) dbHandler {
	return func(db *sql.DB, writer http.ResponseWriter, request *http.Request) {
		extensionId := chi.URLParam(request, "extensionId")
		version := chi.URLParam(request, "extensionVersion")
		err := apiContext.Controller.UninstallExtension(request.Context(), db, extensionId, version)
		if err != nil {
			HandleError(request.Context(), writer, err)
			return
		}
		SendNoContent(request.Context(), writer)
	}
}

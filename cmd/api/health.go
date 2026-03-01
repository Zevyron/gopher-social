package main

import (
	"net/http"
)

// healthCheckHandler godoc
//
//	@Summary		Health check
//	@Description	Returns current service status, environment, and application version
//	@Tags			ops
//	@Produce		json
//	@Success		200	{object}	map[string]string	"Service health status"
//	@Failure		500	{object}	map[string]any		"Internal server error"
//	@Router			/health [get]
func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status":  "ok",
		"env":    	app.config.env,
		"version": 	version,
	}

	if err := writeJSON(w, http.StatusOK, data); err != nil {
		app.internalServerError(w, r, err)
	}
}
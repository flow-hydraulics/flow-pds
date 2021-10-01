package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Create a distribution
func HandleCreateDistribution(logger *log.Logger, app *app.App) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// Check body is not empty
		if err := checkNonEmptyBody(r); err != nil {
			handleError(rw, logger, err)
			return
		}

		var reqDist ReqCreateDistribution

		// Decode JSON
		if err := json.NewDecoder(r.Body).Decode(&reqDist); err != nil {
			handleError(rw, logger, err)
			return
		}

		// Create new distribution
		appDist := reqDist.ToApp()
		if err := app.CreateDistribution(r.Context(), &appDist); err != nil {
			handleError(rw, logger, err)
			return
		}

		res := ResCreateDistribution{
			ID:     appDist.ID,
			FlowID: appDist.FlowID,
		}

		handleJsonResponse(rw, http.StatusCreated, res)
	}
}

// List distributions
func HandleListDistributions(logger *log.Logger, app *app.App) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		limit, err := strconv.Atoi(r.FormValue("limit"))
		if err != nil {
			limit = 0
		}

		offset, err := strconv.Atoi(r.FormValue("offset"))
		if err != nil {
			offset = 0
		}

		list, err := app.ListDistributions(r.Context(), limit, offset)
		if err != nil {
			handleError(rw, logger, err)
			return
		}

		res := ResDistributionListFromApp(list)

		handleJsonResponse(rw, http.StatusOK, res)
	}
}

// Get distribution details
func HandleGetDistribution(logger *log.Logger, app *app.App) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		id, err := uuid.Parse(vars["id"])
		if err != nil {
			handleError(rw, logger, err)
			return
		}

		dist, err := app.GetDistribution(r.Context(), id)
		if err != nil {
			handleError(rw, logger, err)
			return
		}

		res := ResGetDistributionFromApp(dist)

		handleJsonResponse(rw, http.StatusOK, res)
	}
}

// Cancel a distribution
func HandleCancelDistribution(logger *log.Logger, app *app.App) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		id, err := uuid.Parse(vars["id"])
		if err != nil {
			handleError(rw, logger, err)
			return
		}

		if err := app.CancelDistribution(r.Context(), id); err != nil {
			handleError(rw, logger, err)
			return
		}

		handleJsonResponse(rw, http.StatusOK, "Ok")
	}
}

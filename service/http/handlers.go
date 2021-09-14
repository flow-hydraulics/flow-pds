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

		var dist ReqCreateDistribution

		// Decode JSON
		if err := json.NewDecoder(r.Body).Decode(&dist); err != nil {
			handleError(rw, logger, err)
			return
		}

		// Create new distribution
		id, err := app.CreateDistribution(dist.ToApp())
		if err != nil {
			handleError(rw, logger, err)
			return
		}

		res := ResCreateDistribution{DistributionId: id}

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

		list, err := app.ListDistributions(limit, offset)
		if err != nil {
			handleError(rw, logger, err)
			return
		}

		res := make([]ResDistributionListItem, len(list))
		for i := range res {
			res[i] = ResDistributionListItemFromApp(list[i])
		}

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

		dist, err := app.GetDistribution(id)
		if err != nil {
			handleError(rw, logger, err)
			return
		}

		res := ResDistributionFromApp(*dist)

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

		err = app.CancelDistribution(id)
		if err != nil {
			handleError(rw, logger, err)
			return
		}

		handleJsonResponse(rw, http.StatusOK, "Ok")
	}
}

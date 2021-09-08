package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/flow-hydraulics/flow-pds/service/app"
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

		var dist CreateDistributionRequest

		// Decode JSON
		if err := json.NewDecoder(r.Body).Decode(&dist); err != nil {
			handleError(rw, logger, fmt.Errorf("invalid body"))
			return
		}

		// Create new distribution
		id, err := app.CreateDistribution(dist.ToApp())
		if err != nil {
			handleError(rw, logger, err)
			return
		}

		res := CreateDistributionResponse{DistributionId: id}

		handleJsonResponse(rw, http.StatusCreated, res)
	}
}

// List distributions
func HandleListDistributions(logger *log.Logger, app *app.App) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		list, err := app.ListDistributions()
		if err != nil {
			handleError(rw, logger, err)
		}

		res := make([]Distribution, len(list))
		for i := range res {
			res[i] = DistributionFromApp(list[i])
		}

		handleJsonResponse(rw, http.StatusOK, res)
	}
}

// Get distribution details
func HandleGetDistribution(logger *log.Logger, app *app.App) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		dist, err := app.GetDistribution(vars["id"])
		if err != nil {
			handleError(rw, logger, err)
		}

		res := DistributionFromApp(*dist)

		handleJsonResponse(rw, http.StatusOK, res)
	}
}

// Settle a distribution
func HandleSettleDistribution(logger *log.Logger, app *app.App) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		err := app.SettleDistribution(vars["id"])
		if err != nil {
			handleError(rw, logger, err)
		}

		handleJsonResponse(rw, http.StatusOK, "Ok")
	}
}

// Confirm a distribution
func HandleConfirmDistribution(logger *log.Logger, app *app.App) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		err := app.ConfirmDistribution(vars["id"])
		if err != nil {
			handleError(rw, logger, err)
		}

		handleJsonResponse(rw, http.StatusOK, "Ok")
	}
}

// Cancel a distribution
func HandleCancelDistribution(logger *log.Logger, app *app.App) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		err := app.CancelDistribution(vars["id"])
		if err != nil {
			handleError(rw, logger, err)
		}

		handleJsonResponse(rw, http.StatusOK, "Ok")
	}
}

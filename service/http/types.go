package http

import (
	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/onflow/flow-go-sdk"
)

type CreateDistributionResponse struct {
	// TODO (latenssi)
	DistributionId string `json:"distributionId"`
}

type CreatDistributionRequest struct {
	// TODO (latenssi)
	Issuer string `json:"issuer"`
}

type Distribution struct {
	// TODO (latenssi)
	Issuer string `json:"issuer"`
}

func DistributionFromApp(appDist app.Distribution) Distribution {
	// TODO (latenssi)
	return Distribution{
		Issuer: appDist.Issuer.Hex(),
	}
}

func (d *CreatDistributionRequest) ToApp() app.Distribution {
	// TODO (latenssi)
	return app.Distribution{
		Issuer: flow.HexToAddress(d.Issuer),
	}
}

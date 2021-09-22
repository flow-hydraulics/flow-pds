package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flow-hydraulics/flow-pds/service/common"
	pds_http "github.com/flow-hydraulics/flow-pds/service/http"
	"github.com/onflow/flow-go-sdk"
)

func benchmarkCreate(packs, slots uint, b *testing.B) {
	cfg := getTestCfg()
	server, cleanup := getTestServer(cfg)
	defer func() {
		cleanup()
	}()

	addr := common.FlowAddress(flow.HexToAddress("0x1"))
	collection := makeTestCollection(int(packs * slots))

	dReq := pds_http.ReqCreateDistribution{
		DistID: common.FlowID{Int64: int64(1), Valid: true},
		Issuer: addr,
		PackTemplate: pds_http.PackTemplate{
			PackReference: pds_http.AddressLocation{
				Name:    "TestPackNFT",
				Address: addr,
			},
			PackCount: packs,
			Buckets: []pds_http.Bucket{
				{
					CollectibleReference: pds_http.AddressLocation{
						Name:    "TestCollectibleNFT",
						Address: addr,
					},
					CollectibleCount:      slots,
					CollectibleCollection: collection,
				},
			},
		},
	}

	jReq, err := json.Marshal(dReq)
	if err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		req, err := http.NewRequest("POST", "/v1/distributions", bytes.NewBuffer(jReq))
		if err != nil {
			b.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		server.Server.Handler.ServeHTTP(rr, req)

		// Check the status code is what we expect.
		if status := rr.Code; status != http.StatusCreated {
			b.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
			b.Log(rr.Body)
		}
	}
}

func BenchmarkCreate1Collectible(b *testing.B)     { benchmarkCreate(1, 1, b) }
func BenchmarkCreate1kCollectibles(b *testing.B)   { benchmarkCreate(100, 10, b) }
func BenchmarkCreate10kCollectibles(b *testing.B)  { benchmarkCreate(1000, 10, b) }
func BenchmarkCreate20kCollectibles(b *testing.B)  { benchmarkCreate(2000, 10, b) }
func BenchmarkCreate30kCollectibles(b *testing.B)  { benchmarkCreate(3000, 10, b) }
func BenchmarkCreate40kCollectibles(b *testing.B)  { benchmarkCreate(4000, 10, b) }
func BenchmarkCreate50kCollectibles(b *testing.B)  { benchmarkCreate(5000, 10, b) }
func BenchmarkCreate100kCollectibles(b *testing.B) { benchmarkCreate(10000, 10, b) }

// func BenchmarkCreate1MCollectibles(b *testing.B)   { benchmarkCreate(100000, 10, b) }

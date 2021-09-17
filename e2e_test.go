package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flow-hydraulics/flow-pds/service/app"
	"github.com/flow-hydraulics/flow-pds/service/common"
	pds_http "github.com/flow-hydraulics/flow-pds/service/http"
	"github.com/google/uuid"
	"github.com/onflow/flow-go-sdk"
)

func TestCreate(t *testing.T) {
	cfg := getTestCfg()
	server, cleanup := getTestServer(cfg)
	defer func() {
		cleanup()
	}()

	packs := 10
	slotsPerBucket := 5

	addr := common.FlowAddress(flow.HexToAddress("0x1"))
	collection := makeTestCollection(packs * slotsPerBucket)

	dReq := pds_http.ReqCreateDistribution{
		DistID: 1,
		Issuer: addr,
		PackTemplate: pds_http.PackTemplate{
			PackReference: pds_http.AddressLocation{
				Name:    "TestPackNFT",
				Address: addr,
			},
			PackCount: uint(packs),
			Buckets: []pds_http.Bucket{
				{
					CollectibleReference: pds_http.AddressLocation{
						Name:    "TestCollectibleNFT",
						Address: addr,
					},
					CollectibleCount:      uint(slotsPerBucket),
					CollectibleCollection: collection,
				},
			},
		},
	}

	jReq, err := json.Marshal(dReq)
	if err != nil {
		t.Fatal(err)
	}

	// Create

	rr1 := httptest.NewRecorder()

	createReq, err := http.NewRequest("POST", "/v1/distributions", bytes.NewBuffer(jReq))
	if err != nil {
		t.Fatal(err)
	}

	createReq.Header.Set("Content-Type", "application/json")

	server.Server.Handler.ServeHTTP(rr1, createReq)

	// Check the status code is what we expect.
	if status := rr1.Code; status != http.StatusCreated {
		t.Fatalf("handler returned wrong status code: got %v want %v, error: %s", status, http.StatusCreated, rr1.Body)
	}

	createRes := pds_http.ResCreateDistribution{}
	if err := json.NewDecoder(rr1.Body).Decode(&createRes); err != nil {
		t.Fatal(err)
	}

	AssertNotEqual(t, createRes.DistributionId, uuid.Nil)

	// Get

	rr2 := httptest.NewRecorder()

	getReq, err := http.NewRequest("GET", fmt.Sprintf("/v1/distributions/%s", createRes.DistributionId), nil)
	if err != nil {
		t.Fatal(err)
	}

	server.Server.Handler.ServeHTTP(rr2, getReq)

	// Check the status code is what we expect.
	if status := rr2.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v, error: %s", status, http.StatusOK, rr2.Body)
	}

	getRes := pds_http.ResDistribution{}
	if err := json.NewDecoder(rr2.Body).Decode(&getRes); err != nil {
		t.Fatal(err)
	}

	AssertEqual(t, getRes.ID, createRes.DistributionId)
	AssertEqual(t, getRes.Issuer, addr)
	AssertEqual(t, len(getRes.ResolvedCollection), len(collection))
	AssertEqual(t, len(getRes.Packs), packs)
	AssertEqual(t, getRes.Packs[0].CommitmentHash.IsEmpty(), false)
}

func TestStartSettlement(t *testing.T) {
	cfg := getTestCfg()
	a, cleanup := getTestApp(cfg)
	defer func() {
		cleanup()
	}()

	addr := common.FlowAddress(flow.HexToAddress("0x1"))
	collection := makeTestCollection(10)

	d := app.Distribution{
		DistID: 1,
		Issuer: addr,
		PackTemplate: app.PackTemplate{
			PackReference: app.AddressLocation{
				Name:    "TestPackNFT",
				Address: addr,
			},
			PackCount: 2,
			Buckets: []app.Bucket{
				{
					CollectibleReference: app.AddressLocation{
						Name:    "TestCollectibleNFT",
						Address: addr,
					},
					CollectibleCount:      5,
					CollectibleCollection: collection,
				},
			},
		},
	}

	if err := a.CreateDistribution(context.Background(), &d); err != nil {
		t.Fatal(err)
	}

	if err := a.SettleDistribution(context.Background(), d.ID); err != nil {
		t.Fatal(err)
	}

	_, settlement, err := a.GetDistribution(context.Background(), d.ID)
	if err != nil {
		t.Fatal(err)
	}

	if settlement == nil {
		t.Fatal("expected settlement to exist")
	}

	AssertNotEqual(t, settlement.ID, uuid.Nil)
	AssertEqual(t, settlement.Settled, uint(0))
	AssertEqual(t, settlement.Total, uint(len(collection)))

	// Try to start settlement again
	if err := a.SettleDistribution(context.Background(), d.ID); err == nil {
		t.Fatal("expected an error")
	}
}

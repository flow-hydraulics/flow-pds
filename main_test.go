package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bjartek/go-with-the-flow/v2/gwtf"
	"github.com/flow-hydraulics/flow-pds/go-contracts/util"
	"github.com/flow-hydraulics/flow-pds/service/common"
	pds_http "github.com/flow-hydraulics/flow-pds/service/http"
	"github.com/google/uuid"
	"github.com/onflow/flow-go-sdk"
)

func TestCreate(t *testing.T) {
	t.Skip("focusing on e2e test now")

	cfg := getTestCfg(t, nil)
	server, cleanup := getTestServer(cfg, false)
	defer func() {
		cleanup()
	}()

	packs := 10
	slotsPerBucket := 5

	addr := common.FlowAddress(flow.HexToAddress("0x1"))
	collection := makeTestCollection(packs * slotsPerBucket)

	dReq := pds_http.ReqCreateDistribution{
		FlowID: common.FlowID{Int64: int64(1), Valid: true},
		Issuer: addr,
		PackTemplate: pds_http.ReqPackTemplate{
			PackReference: pds_http.AddressLocation{
				Name:    "TestPackNFT",
				Address: addr,
			},
			CollectibleReference: pds_http.AddressLocation{
				Name:    "TestCollectibleNFT",
				Address: addr,
			},
			PackCount: uint(packs),
			Buckets: []pds_http.ReqBucket{
				{
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

	AssertNotEqual(t, createRes.ID, uuid.Nil)

	// Get

	rr2 := httptest.NewRecorder()

	getReq, err := http.NewRequest("GET", fmt.Sprintf("/v1/distributions/%s", createRes.ID), nil)
	if err != nil {
		t.Fatal(err)
	}

	server.Server.Handler.ServeHTTP(rr2, getReq)

	// Check the status code is what we expect.
	if status := rr2.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v, error: %s", status, http.StatusOK, rr2.Body)
	}

	getRes := pds_http.ResGetDistribution{}
	if err := json.NewDecoder(rr2.Body).Decode(&getRes); err != nil {
		t.Fatal(err)
	}

	AssertEqual(t, getRes.ID, createRes.ID)
	AssertEqual(t, getRes.Issuer, addr)
}

func TestSetDistCap(t *testing.T) {
	cfg := getTestCfg(t, nil)
	server, cleanup := getTestServer(cfg, false)
	defer func() {
		cleanup()
	}()

	g := gwtf.NewGoWithTheFlow([]string{"./flow.json"}, "emulator", false, 0)

	t.Log("Issuer create PackIssuer resource to store DistCap")

	createPackIssuer := "./cadence-transactions/pds/create_new_pack_issuer.cdc"
	createPackIssuerCode := util.ParseCadenceTemplate(createPackIssuer)
	_, err := g.
		TransactionFromFile(createPackIssuer, createPackIssuerCode).
		SignProposeAndPayAs("issuer").
		RunE()
	if err != nil {
		t.Fatal(err)
	}

	issuer := common.FlowAddress(g.Account("issuer").Address())

	time.Sleep(cfg.TransactionPollInterval * 2)

	dReq := pds_http.ReqCreateDistribution{
		Issuer: issuer,
	}

	jReq, err := json.Marshal(dReq)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRecorder()

	req, err := http.NewRequest("POST", "/v1/set-dist-cap", bytes.NewBuffer(jReq))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	server.Server.Handler.ServeHTTP(r, req)

	// Check the status code is what we expect.
	if status := r.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v, error: %s", status, http.StatusOK, r.Body)
	}
}

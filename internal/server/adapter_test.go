package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/and161185/metrics-alerting/internal/errs"
	"github.com/and161185/metrics-alerting/storage/inmemory"
)

func newAdapter(t *testing.T) (context.Context, storageAdapter) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancel)
	st := inmemory.NewMemStorage(ctx)
	return ctx, storageAdapter{s: st}
}

func TestStorageAdapter_Ping_OK(t *testing.T) {
	ctx, ad := newAdapter(t)
	if err := ad.Ping(ctx); err != nil {
		t.Fatalf("Ping() err = %v", err)
	}
}

func TestStorageAdapter_UpdateGauge_GetGauge(t *testing.T) {
	ctx, ad := newAdapter(t)

	// not found
	if _, ok, err := ad.GetGauge(ctx, "g1"); err == nil || !errors.Is(err, errs.ErrMetricNotFound) || ok {
		t.Fatalf("GetGauge not-found: err=%v ok=%v", err, ok)
	}

	// set value
	if err := ad.UpdateGauge(ctx, "g1", 1.23); err != nil {
		t.Fatalf("UpdateGauge err=%v", err)
	}
	v, ok, err := ad.GetGauge(ctx, "g1")
	if err != nil {
		t.Fatalf("GetGauge err=%v", err)
	}
	if !ok || v != 1.23 {
		t.Fatalf("GetGauge got ok=%v v=%v", ok, v)
	}

	// overwrite gauge
	if err := ad.UpdateGauge(ctx, "g1", 7.5); err != nil {
		t.Fatalf("UpdateGauge overwrite err=%v", err)
	}
	v2, ok2, err := ad.GetGauge(ctx, "g1")
	if err != nil || !ok2 || v2 != 7.5 {
		t.Fatalf("GetGauge overwrite got v=%v ok=%v err=%v", v2, ok2, err)
	}
}

func TestStorageAdapter_UpdateCounter_GetCounter(t *testing.T) {
	ctx, ad := newAdapter(t)

	// not found
	if _, ok, err := ad.GetCounter(ctx, "c1"); err == nil || !errors.Is(err, errs.ErrMetricNotFound) || ok {
		t.Fatalf("GetCounter not-found: err=%v ok=%v", err, ok)
	}

	// add deltas (accumulate)
	if err := ad.UpdateCounter(ctx, "c1", 5); err != nil {
		t.Fatalf("UpdateCounter #1 err=%v", err)
	}
	if err := ad.UpdateCounter(ctx, "c1", 2); err != nil {
		t.Fatalf("UpdateCounter #2 err=%v", err)
	}
	v, ok, err := ad.GetCounter(ctx, "c1")
	if err != nil {
		t.Fatalf("GetCounter err=%v", err)
	}
	if !ok || v != 7 {
		t.Fatalf("GetCounter got ok=%v v=%v", ok, v)
	}
}

func TestStorageAdapter_GetAll(t *testing.T) {
	ctx, ad := newAdapter(t)

	// seed
	if err := ad.UpdateGauge(ctx, "gA", 3.14); err != nil {
		t.Fatalf("UpdateGauge err=%v", err)
	}
	if err := ad.UpdateCounter(ctx, "cA", 10); err != nil {
		t.Fatalf("UpdateCounter #1 err=%v", err)
	}
	if err := ad.UpdateCounter(ctx, "cA", 5); err != nil {
		t.Fatalf("UpdateCounter #2 err=%v", err)
	}

	gs, cs, err := ad.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll err=%v", err)
	}
	if len(gs) != 1 || len(cs) != 1 {
		t.Fatalf("GetAll sizes: gauges=%d counters=%d", len(gs), len(cs))
	}
	if gs["gA"] != 3.14 {
		t.Fatalf("gauges[gA]=%v", gs["gA"])
	}
	if cs["cA"] != 15 {
		t.Fatalf("counters[cA]=%v", cs["cA"])
	}
}

package models

import (
	"testing"

	"github.com/meedamian/fat/internal/types"
)

func TestCostForToks(t *testing.T) {
	mi := &types.ModelInfo{
		Rates: types.Rate{In: 0.1, Out: 0.2},
	}
	cost := CostForToks(mi, 10, 20)
	expected := 0.1*10 + 0.2*20
	if cost != expected {
		t.Errorf("CostForToks = %f; want %f", cost, expected)
	}
}

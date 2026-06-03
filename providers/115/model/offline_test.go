package model

import (
	"testing"

	"github.com/onetreegrove/panctl/pkg/contract"
)

func TestOfflineTaskToContract(t *testing.T) {
	got := OfflineTask{GID: "g", Name: "demo.mp4", StatusText: "running", Progress: 42.5}.ToContract()
	if got.GID != "g" || got.Status != contract.OfflineRunning || got.Progress != 42.5 {
		t.Fatalf("task = %+v", got)
	}
}

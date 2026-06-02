package baidu

import "testing"

func TestProviderCapabilitiesAreConservativeButUseful(t *testing.T) {
	caps := New().Capabilities()

	if !caps.PathLookup || !caps.List || !caps.Mkdir || !caps.Rename || !caps.Move || !caps.Copy || !caps.Remove || !caps.Download || !caps.Upload || !caps.RapidUpload {
		t.Fatalf("expected baidu file capabilities to be enabled: %+v", caps)
	}
	if caps.Search || caps.OfflineTask || caps.Share || caps.RecycleBin || caps.CrossTransfer {
		t.Fatalf("expected unsupported baidu capabilities to remain disabled: %+v", caps)
	}
}

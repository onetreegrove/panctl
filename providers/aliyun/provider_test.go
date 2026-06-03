package aliyun

import "testing"

func TestProviderCapabilities(t *testing.T) {
	caps := New().Capabilities()
	if !caps.PathLookup || !caps.List || !caps.Mkdir || !caps.Rename || !caps.Move || !caps.Copy || !caps.Remove || !caps.Download || !caps.Upload || !caps.RapidUpload {
		t.Fatalf("expected aliyun file capabilities to be enabled: %+v", caps)
	}
	if caps.Search || caps.OfflineTask || caps.Share || caps.RecycleBin || caps.CrossTransfer {
		t.Fatalf("expected unsupported aliyun capabilities to remain disabled: %+v", caps)
	}
	if New().Name() != "aliyun" {
		t.Fatalf("unexpected provider name: %s", New().Name())
	}
}

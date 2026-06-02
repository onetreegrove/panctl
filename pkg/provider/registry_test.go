package provider

import "testing"

type fakeProvider struct{}

func (fakeProvider) Name() string { return "fake" }
func (fakeProvider) Capabilities() Capabilities {
	return Capabilities{List: true}
}

func TestRegistryReturnsProvider(t *testing.T) {
	reg := NewRegistry()
	reg.Register(fakeProvider{})
	got, ok := reg.Get("fake")
	if !ok {
		t.Fatal("provider not found")
	}
	if got.Name() != "fake" {
		t.Fatalf("name = %q", got.Name())
	}
}

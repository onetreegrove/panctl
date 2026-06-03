package pan115

import "github.com/onetreegrove/panctl/pkg/provider"

type Provider struct{}

func New() Provider {
	return Provider{}
}

func (Provider) Name() string {
	return "115"
}

func (Provider) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		PathLookup:  true,
		List:        true,
		Mkdir:       true,
		Rename:      true,
		Move:        true,
		Copy:        true,
		Remove:      true,
		Download:    true,
		OfflineTask: true,
		Upload:      true, // Enable upload capability
		Share:       true, // Enable share capability
	}
}

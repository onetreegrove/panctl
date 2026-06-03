package baidu

import "github.com/onetreegrove/panctl/pkg/provider"

type Provider struct{}

func New() Provider {
	return Provider{}
}

func (Provider) Name() string {
	return "baidu"
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
		Upload:      true,
		RapidUpload: true,
	}
}

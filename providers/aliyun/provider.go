package aliyun

import "github.com/justonetree/pan-cli/pkg/provider"

type Provider struct{}

func New() Provider {
	return Provider{}
}

func (Provider) Name() string {
	return "aliyun"
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

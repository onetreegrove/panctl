package provider

type Capabilities struct {
	PathLookup    bool `json:"path_lookup"`
	List          bool `json:"list"`
	Search        bool `json:"search"`
	Mkdir         bool `json:"mkdir"`
	Rename        bool `json:"rename"`
	Move          bool `json:"move"`
	Copy          bool `json:"copy"`
	Remove        bool `json:"remove"`
	Download      bool `json:"download"`
	Upload        bool `json:"upload"`
	OfflineTask   bool `json:"offline_task"`
	Share         bool `json:"share"`
	RecycleBin    bool `json:"recycle_bin"`
	RapidUpload   bool `json:"rapid_upload"`
	CrossTransfer bool `json:"cross_transfer"`
}

type Provider interface {
	Name() string
	Capabilities() Capabilities
}

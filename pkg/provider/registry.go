package provider

type Registry struct {
	items map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{items: map[string]Provider{}}
}

func (r *Registry) Register(p Provider) {
	r.items[p.Name()] = p
}

func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.items[name]
	return p, ok
}

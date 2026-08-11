package health

type Repository struct {
	serviceName string
	environment string
}

func NewRepository(serviceName, environment string) Repository {
	return Repository{serviceName: serviceName, environment: environment}
}

func (r Repository) Metadata() (string, string) {
	return r.serviceName, r.environment
}

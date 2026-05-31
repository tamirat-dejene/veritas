package providerregistry

import (
	"fmt"

	"github.com/tamirat-dejene/veritas/services/payment-service/internal/domain"
)

type ProviderRegistry struct {
	providers map[string]domain.PaymentProvider
}

func NewProviderRegistry(stripe, chapa domain.PaymentProvider) *ProviderRegistry {
	return &ProviderRegistry{
		providers: map[string]domain.PaymentProvider{
			domain.PaymentProviderStripe: stripe,
			domain.PaymentProviderChapa:  chapa,
		},
	}
}

func (r *ProviderRegistry) Get(name string) (domain.PaymentProvider, error) {
	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidProvider, name)
	}
	return provider, nil
}

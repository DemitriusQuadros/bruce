package handlers

import (
	"net/http"

	"bruce/internal/ai"
)

// providerInfo represents a provider's availability and configuration.
type providerInfo struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Model     string `json:"model"`
}

// ProvidersHandler godoc
// @Summary      List available LLM providers
// @Description  Returns a list of LLM providers and their availability
// @Tags         providers
// @Produce      json
// @Success      200  {object}  []providerInfo
// @Router       /api/v1/providers [get]
func ProvidersHandler(registry *ai.ProviderRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		info := registry.ListProviders()
		writeJSON(w, http.StatusOK, info)
	}
}

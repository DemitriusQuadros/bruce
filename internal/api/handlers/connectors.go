package handlers

import (
	"net/http"

	"bruce/internal/worker"
)

// connectorResponse represents the status of a connector.
type connectorResponse struct {
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"` // "connected" | "disconnected" | "needs_qr" | "disabled"
}

// ConnectorsHandler godoc
// @Summary      Get connector status
// @Description  Returns the status of all connectors (WhatsApp, Discord)
// @Tags         connectors
// @Produce      json
// @Success      200  {object}  []connectorResponse
// @Router       /api/v1/connectors [get]
func ConnectorsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry, ok := r.Context().Value("dispatcherRegistry").(*worker.DispatcherRegistry)
		if !ok {
			// If registry is not available, return empty list with basic status.
			result := []connectorResponse{
				{Type: "whatsapp", Enabled: false, Status: "disabled"},
				{Type: "discord", Enabled: false, Status: "disabled"},
				{Type: "telegram", Enabled: false, Status: "disabled"},
			}
			writeJSON(w, http.StatusOK, result)
			return
		}

		// Check which connectors are registered.
		whatsappEnabled := registry.Get("whatsapp") != nil
		discordEnabled := registry.Get("discord") != nil
		telegramEnabled := registry.Get("telegram") != nil

		result := []connectorResponse{
			{
				Type:    "whatsapp",
				Enabled: whatsappEnabled,
				Status:  getStatus(whatsappEnabled),
			},
			{
				Type:    "discord",
				Enabled: discordEnabled,
				Status:  getStatus(discordEnabled),
			},
			{
				Type:    "telegram",
				Enabled: telegramEnabled,
				Status:  getStatus(telegramEnabled),
			},
		}

		writeJSON(w, http.StatusOK, result)
	}
}

// getStatus returns a connector status string based on its enabled state.
// In Phase 1, we only track "connected" or "disabled" since connector state
// management is limited. More detailed status (needs_qr, disconnected) will
// be added in Phase 2 when we implement live state tracking.
func getStatus(enabled bool) string {
	if enabled {
		return "connected"
	}
	return "disabled"
}

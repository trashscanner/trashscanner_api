package api

import (
	"net/http"
	"net/netip"
	"strings"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func (s *Server) writeLoginHistory(r *http.Request, statusCode int, err error) {
	u, ok := utils.GetUser(r.Context()).(models.User)
	if !ok || u.ID == uuid.Nil {
		return
	}

	ipStr := r.Header.Get("X-Real-IP")
	if ipStr == "" {
		ipStr = r.Header.Get("X-Forwarded-For")
		if ipStr != "" {
			ipStr = strings.Split(ipStr, ",")[0]
		}
	}
	if ipStr == "" {
		ipStr = r.RemoteAddr
		if idx := strings.LastIndex(ipStr, ":"); idx != -1 {
			ipStr = ipStr[:idx]
		}
	}

	var ipAddr *netip.Addr
	if parsedIP, err := netip.ParseAddr(ipStr); err == nil {
		ipAddr = &parsedIP
	}

	location := r.Header.Get("X-Location")
	var locationPtr *string
	if location != "" {
		locationPtr = &location
	}

	userAgent := r.UserAgent()
	var userAgentPtr *string
	if userAgent != "" {
		userAgentPtr = &userAgent
	}

	loginHistory := &models.LoginHistory{
		UserID:    u.ID,
		Success:   statusCode >= 200 && statusCode < 300,
		IpAddress: ipAddr,
		UserAgent: userAgentPtr,
		Location:  locationPtr,
	}
	if err != nil {
		str := err.Error()
		loginHistory.FailureReason = &str
	}

_ = s.store.InsertLoginHistory(r.Context(), loginHistory)
}
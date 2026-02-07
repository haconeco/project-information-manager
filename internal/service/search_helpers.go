package service

import (
	"strings"

	"github.com/haconeco/project-information-manager/internal/domain"
)

func matchesQuery(query string, fields ...string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), q) {
			return true
		}
	}
	return false
}

func joinTags(tags []string) string {
	return strings.Join(tags, " ")
}

func priorityWeight(priority domain.Priority) float32 {
	switch priority {
	case domain.PriorityP0:
		return 1.30
	case domain.PriorityP1:
		return 1.15
	case domain.PriorityP2:
		return 1.00
	case domain.PriorityP3:
		return 0.85
	default:
		return 1.00
	}
}

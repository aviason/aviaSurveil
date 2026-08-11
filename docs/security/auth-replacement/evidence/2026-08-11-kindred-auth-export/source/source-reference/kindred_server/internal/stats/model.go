package stats

import "time"

type MeStats struct {
	PointsAvailable   int       `json:"pointsAvailable"`
	ItemsListed       int       `json:"itemsListed"`
	ItemsActive       int       `json:"itemsActive"`
	ItemsCompleted    int       `json:"itemsCompleted"`
	RequestsOpen      int       `json:"requestsOpen"`
	RequestsCompleted int       `json:"requestsCompleted"`
	City              string    `json:"city,omitempty"`
	GeneratedAt       time.Time `json:"generatedAt"`
}

type CommunityStats struct {
	AvailableItems int       `json:"availableItems"`
	ActiveDonors   int       `json:"activeDonors"`
	RadiusKm       float64   `json:"radiusKm"`
	GeneratedAt    time.Time `json:"generatedAt"`
}

type CommunityQuery struct {
	Lat      float64
	Lng      float64
	RadiusKm float64
}

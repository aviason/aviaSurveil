package health

import "time"

type Status struct {
	Service     string    `json:"service"`
	Environment string    `json:"environment"`
	Status      string    `json:"status"`
	Time        time.Time `json:"time"`
}

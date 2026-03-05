package models

type Artifact struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Lat           float64 `json:"lat"`
	Lng           float64 `json:"lng"`
	Category      string  `json:"category"`
	CreatedBy     int     `json:"created_by"`
	PhotoKey      string  `json:"-"`
	PhotoURL      string  `json:"photo_url,omitempty"`
	Confirmations int     `json:"confirmations"`
	HasConfirmed  bool    `json:"has_confirmed"`
}

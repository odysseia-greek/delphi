package models

import "encoding/json"

// swagger:model
type SolonResponse struct {
	// example: true
	// required: true
	Created bool `json:"created"`
}

func (r *SolonResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func (r *TokenResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// swagger:model
type TokenResponse struct {
	// example: s.0982371293fj
	// required: true
	Token string `json:"token"`
}

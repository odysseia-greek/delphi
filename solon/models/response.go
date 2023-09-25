package models

import "encoding/json"

func (r *TokenResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// swagger:model
type TokenResponse struct {
	// example: s.0982371293fj
	// required: true
	Token string `json:"token"`
}

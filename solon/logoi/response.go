package logoi

import "encoding/json"

func (r *TokenResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type TokenResponse struct {
	// example: s.0982371293fj
	// required: true
	Token string `json:"token"`
}

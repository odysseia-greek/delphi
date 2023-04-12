package models

import "encoding/json"

type SolonResponse struct {
	Created bool `json:"created"`
}

func (r *SolonResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func (r *TokenResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type TokenResponse struct {
	Token string `json:"token"`
}

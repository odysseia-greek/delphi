package models

import "encoding/json"

func UnmarshalSolonCreationRequest(data []byte) (SolonCreationRequest, error) {
	var r SolonCreationRequest
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *SolonCreationRequest) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// swagger:model
type SolonCreationRequest struct {
	// example: api
	// required: true
	Role string `json:"roles"`
	// example: ["dictionary"]
	// required: true
	Access []string `json:"access"`
	// example: alexandros-79bbf86f4b-s48lc
	// required: true
	PodName string `json:"podName"`
	// example: alexandros
	// required: true
	Username string `json:"username"`
}

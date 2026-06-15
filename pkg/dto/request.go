package dto

// TitlesDTO  represent required titles
// swagger:model TitlesDTO
type TitlesDTO struct {
	// array of required titles
	// required: true
	// example: ["btc", "eth"]
	Titles []string `json:"titles"`
}

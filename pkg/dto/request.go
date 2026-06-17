package dto

// TitlesDTO represents request with coin titles.
// swagger:model TitlesDTO
type TitlesDTO struct {
	// coin titles
	// required: true
	Titles []string `json:"titles" example:"btc,eth"`
}

package common

// Response in the response structure of api setu
type Response struct {
	Sessions []Session `json:"sessions"`
}

// Session is the nested structure of api setu response
type Session struct {
	CenterID     int `json:"center_id"`
	Name         string
	Address      string
	StateName    string `json:"state_name"`
	DistrictName string `json:"district_name"`
	date         string
	Capacity     int `json:"available_capacity"`
	AgeLimit     int `json:"min_age_limit"`
	vaccine      string
	slots        []string
}

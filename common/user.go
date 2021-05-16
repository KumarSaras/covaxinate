package common

import "github.com/slack-go/slack"

/*User is a user struct*/
type User struct {
	ID       string `json:"user_id"`
	District int    `json:"district_id"`
	Vaccine  string `json:"vaccine"`
	MinAge   string `json:"min_age"`
}

// NewUser team join event struct
type NewUser struct {
	Event slack.TeamJoinEvent
}

// ChallengeMessage is struct for slack verification
type ChallengeMessage struct {
	Challenge string
}

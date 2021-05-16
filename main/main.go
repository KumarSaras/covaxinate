package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/KumarSaras/covaxinate/common"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"log"
	"os"
	"path/filepath"
)

var api = slack.New(os.Getenv("SLACK_TOKEN"))

const (
	DistrictDialogCallbackID = "enroll_covaxinate"
	StateActionID            = "static_select-action-states"
)

func main() {

	go common.Poll(pollCallback)

	//For slash commands
	http.HandleFunc("/covaxinate/slash", func(w http.ResponseWriter, r *http.Request) {
		slashCommand, err := slack.SlashCommandParse(r)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		postStatesBlockView(slashCommand.UserID)

	})

	http.HandleFunc("/covaxinate", func(w http.ResponseWriter, r *http.Request) {

		if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			r.ParseForm()
			payloadJSON := r.Form.Get("payload")
			var messageAction slack.InteractionCallback
			unMarshalErr := json.Unmarshal([]byte(payloadJSON), &messageAction)
			if unMarshalErr != nil {
				log.Println(unMarshalErr)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if messageAction.Type == slack.InteractionTypeBlockActions && messageAction.ActionCallback.BlockActions[0].ActionID == StateActionID {
				stateID := messageAction.ActionCallback.BlockActions[0].SelectedOption.Value
				postDistrictBlockView(messageAction.User.ID, stateID, messageAction.TriggerID)
			} else if messageAction.Type == slack.InteractionTypeDialogSubmission && messageAction.CallbackID == DistrictDialogCallbackID {
				userID := messageAction.User.ID
				district := messageAction.DialogSubmissionCallback.Submission["district_selection"]
				districtID, _ := strconv.Atoi(district)
				vaccine := messageAction.DialogSubmissionCallback.Submission["vaccine_selection"]
				minAge := messageAction.DialogSubmissionCallback.Submission["age_selection"]
				user := common.User{
					ID:       userID,
					District: districtID,
					Vaccine:  vaccine,
					MinAge:   minAge,
				}
				//This is done to as slack expects a response to a dialog selection within 3 seconds.
				api.PostMessage(userID, slack.MsgOptionText("Your registration has been intitiated. You would be notified if something goes wrong. :crossed_fingers:", false))
				go common.Register(user, registrationCallback)
			}
		} else {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
			if err != nil {
				log.Printf("Parsing event failed - %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if eventsAPIEvent.Type == slackevents.URLVerification {
				var r *slackevents.ChallengeResponse
				err := json.Unmarshal([]byte(body), &r)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "text")
				w.Write([]byte(r.Challenge))
			}
		}
	})

	//Adding port for heroku
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = ":8080"
	} else {
		port = ":" + port
	}
	if err := http.ListenAndServe(port, nil); err != nil {
		panic(err)
	}
}

func postStatesBlockView(userID string) {
	messageViewJSON, err := ioutil.ReadFile(getJsonTemplateFilePath() + "allstates.json")
	if err != nil {
		log.Printf("Unable to open states template - %v", err)
	}
	var messageTabView slack.View
	err = json.Unmarshal(messageViewJSON, &messageTabView)
	if err != nil {
		log.Printf("Unable to parse states template - %v", err)
	}

	_, _, err = api.PostMessage(userID, slack.MsgOptionBlocks(messageTabView.Blocks.BlockSet...))
	if err != nil {
		log.Printf("Posting states view failed - %v", err)
	}
}

func postDistrictBlockView(userID string, stateID string, triggerID string) {

	districtTemplateFileName := getJsonTemplateFilePath() + stateID + ".json"
	dialogJSON, err := ioutil.ReadFile(districtTemplateFileName)
	if err != nil {
		log.Printf("Unable to open district template - %v", err)
	}

	var dialogView slack.Dialog
	err = json.Unmarshal(dialogJSON, &dialogView)
	if err != nil {
		log.Printf("Unable to parse district template - %v", err)
	}

	dialogView.TriggerID = triggerID
	fmt.Println("Trigger ID - ", triggerID)
	err = api.OpenDialog(triggerID, dialogView)
	if err != nil {
		log.Printf("Unable to open district dialog - %v", err)
	}
}

func registrationCallback(err error, userID string) {
	if err != nil {
		api.PostMessage(userID, slack.MsgOptionText("Your registration was unsuccessful. Please try again. If the issue persists, please send a slack message to @Saras", false))
	}
}

func pollCallback(userID string, response common.Response) {
	if len(response.Sessions) != 0 {

		fieldsSections := make([]*slack.SectionBlock, 0)
		// Header Section
		headerText := slack.NewTextBlockObject("mrkdwn", "Vaccination Slots Available:\n", false, false)
		headerSection := slack.NewSectionBlock(headerText, nil, nil)
		footerText := slack.NewTextBlockObject("mrkdwn", "*<https://selfregistration.cowin.gov.in|Click here to schedule your vaccination>*", false, false)
		footerSection := slack.NewSectionBlock(footerText, nil, nil)

		fieldsSections = append(fieldsSections, headerSection)
		countSessions := 0
		for i, session := range response.Sessions {
			fieldsSections = append(fieldsSections, createSlotResponseMsg(session))
			if countSessions > 39 || i == (len(response.Sessions)-1) {
				log.Printf("Slot Message Limit Exhausted. Total - %v, Count - %v", len(response.Sessions), countSessions)
				fieldsSections = append(fieldsSections, footerSection)
				sendSlotMsg(userID, fieldsSections)
				fieldsSections = make([]*slack.SectionBlock, 0)
				countSessions = 0
			}
			countSessions++
		}
	}
}

func createSlotResponseMsg(session common.Session) *slack.SectionBlock {
	nameField := slack.NewTextBlockObject("mrkdwn", "*Name:*\n"+session.Name, false, false)
	addressField := slack.NewTextBlockObject("mrkdwn", "*Address:*\n"+session.Address, false, false)
	stateField := slack.NewTextBlockObject("mrkdwn", "*State:*\n"+session.StateName, false, false)
	distField := slack.NewTextBlockObject("mrkdwn", "*District:*\n"+session.DistrictName, false, false)
	dateField := slack.NewTextBlockObject("mrkdwn", "*Date:*\n"+session.Date, false, false)
	capacityField := slack.NewTextBlockObject("mrkdwn", "*Capacity:*\n"+strconv.Itoa(session.Capacity), false, false)
	ageField := slack.NewTextBlockObject("mrkdwn", "*Age:*\n"+strconv.Itoa(session.AgeLimit), false, false)
	vaccineField := slack.NewTextBlockObject("mrkdwn", "*Vaccine:*\n"+session.Vaccine, false, false)
	var slotString string
	for _, slot := range session.Slots {
		slotString += slot
		slotString += "|"
	}
	slotsField := slack.NewTextBlockObject("mrkdwn", "*Slots:*\n"+slotString, false, false)

	fieldSlice := make([]*slack.TextBlockObject, 0)
	fieldSlice = append(fieldSlice, nameField)
	fieldSlice = append(fieldSlice, addressField)
	fieldSlice = append(fieldSlice, stateField)
	fieldSlice = append(fieldSlice, distField)
	fieldSlice = append(fieldSlice, dateField)
	fieldSlice = append(fieldSlice, capacityField)
	fieldSlice = append(fieldSlice, ageField)
	fieldSlice = append(fieldSlice, vaccineField)
	fieldSlice = append(fieldSlice, slotsField)

	fieldsSection := slack.NewSectionBlock(nil, fieldSlice, nil)

	return fieldsSection
}

func sendSlotMsg(userID string, fieldsSections []*slack.SectionBlock) {
	var blocks []slack.Block = make([]slack.Block, len(fieldsSections))
	for i, fs := range fieldsSections {
		blocks[i] = fs
	}
	msg := slack.MsgOptionBlocks(blocks...)
	_, _, _, err := api.SendMessage(userID, msg)
	if err != nil {
		log.Printf("Sending slot message failed - %v", err)
	}
}

func getJsonTemplateFilePath() string {
	dir, err := filepath.Abs("./")
	if err != nil {
		fmt.Println(err)
	}
	dir = dir + "/slackutils/jsontemplates/"
	return dir
}

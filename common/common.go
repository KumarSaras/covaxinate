package common

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/robfig/cron"
)

var host = "localhost"
var port = 5432
var user = os.Getenv("DB_USER")
var password = os.Getenv("password")
var dbname = os.Getenv("DB_NAME")

// Register is func
func Register(user User, regCallback func(err error, userID string)) {

	db := openDBConn()
	defer db.Close()
	stmt, err := db.Prepare("insert into user_details(user_id, district_id, slot, min_age, vaccine) values( $1, $2, $3, $4, $5 )")
	if err != nil {
		fmt.Printf("cannot prepare statement - %v", err)
		regCallback(err, user.ID)
	}
	_, err = stmt.Exec(user.ID, user.District, getSlot(), user.MinAge, user.Vaccine)
	if err != nil {
		fmt.Printf("cannot insert user - %v", err)
		regCallback(err, user.ID)
	}
	//getAvailability(strconv.Itoa(user.District), user.MinAge, user.Vaccine)
}

var slot = 0

func getSlot() int {
	if slot%3 == 0 {
		slot = 0
	}
	slotToReturn := slot
	slot++
	return slotToReturn
}

func openDBConn() *sql.DB {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}

	return db
}

func getAvailability(district string, minAge string, vaccine string) (Response, error) {
	client := &http.Client{}
	var customerResponse Response
	customerResponse.Sessions = make([]Session, 0)

	for i := 0; i < 5; i++ {
		currentDate := time.Now().AddDate(0, 0, i)
		date := fmt.Sprintf("%02d-%02d-%d", currentDate.Day(), currentDate.Month(), currentDate.Year())
		reqURL := fmt.Sprintf("https://cdn-api.co-vin.in/api/v2/appointment/sessions/public/findByDistrict?district_id=%v&date=%v", district, date)
		req, _ := http.NewRequest(http.MethodGet, reqURL, nil)
		req.Header.Add("Accept-Language", "hi_IN")
		req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.76 Safari/537.36")
		res, resErr := client.Do(req)
		if resErr != nil {
			return Response{}, resErr
		}

		var jsonResponse Response

		json.NewDecoder(res.Body).Decode(&jsonResponse)

		for _, session := range jsonResponse.Sessions {
			// fmt.Printf("Session - %v - %v - %v - %v", session.AgeLimit, minAge, session.Vaccine, vaccine)
			// fmt.Println()
			if strconv.Itoa(session.AgeLimit) == minAge && (len(vaccine) == 0 || strings.EqualFold(vaccine, session.Vaccine)) {
				customerResponse.Sessions = append(customerResponse.Sessions, session)
			}
		}
	}

	return customerResponse, nil
}

var currentSlot = 0
var pollCallbackFunc func(userID string, response Response)

func pollAvailability() {
	db := openDBConn()
	defer db.Close()
	fmt.Printf("CurrentSlot - %d", currentSlot)
	res, err := db.Query("select district_id, min_age, vaccine, user_id, center_ids from user_details where slot=$1", currentSlot)
	if err != nil {
		panic(err)
	}
	defer res.Close()

	for res.Next() {
		var callbackResponse Response
		var districtID int
		var minAge, vaccine, userID string
		var centerIDs []sql.NullInt64
		res.Scan(&districtID, &minAge, &vaccine, &userID, pq.Array(&centerIDs))
		fmt.Printf("%v - %v - %v - %v - %v", districtID, minAge, vaccine, userID, len(centerIDs))
		fmt.Println()
		jsonResponse, availErr := getAvailability(strconv.Itoa(districtID), minAge, vaccine)
		if availErr != nil {
			panic(err)
		}

		centerIDMap := make(map[int64]bool)
		for _, ci := range centerIDs {
			centerIDMap[ci.Int64] = true
		}
		fmt.Printf("CIMap - %v", len(centerIDMap))
		fmt.Println()
		if len(jsonResponse.Sessions) != 0 {
			if len(centerIDs) == 0 {
				callbackResponse.Sessions = jsonResponse.Sessions
				for _, session := range jsonResponse.Sessions {
					queryString := "update user_details set center_ids = array_append(center_ids, " + strconv.Itoa(session.CenterID) + ") where user_id='" + userID + "'"
					_, e := db.Exec(queryString)
					if e != nil {
						fmt.Printf("DB Error when centerID = 0 - %v", e)
						fmt.Println()
					}
				}
			} else {
				centerIDsList := make([]int64, 0)
				for _, session := range jsonResponse.Sessions {
					centerIDsList = append(centerIDsList, int64(session.CenterID))
					if centerIDMap[int64(session.CenterID)] != true {
						callbackResponse.Sessions = append(callbackResponse.Sessions, session)
						centerIDMap[int64(session.CenterID)] = true
					}
				}
				queryString := "update user_details set center_ids = $1 where user_id=$2"
				_, e := db.Exec(queryString, pq.Array(centerIDsList), userID)
				if e != nil {
					fmt.Printf("DB Error - %v", e)
					fmt.Println()
				}
			}
		} else {
			if len(centerIDs) != 0 {
				emptyCenterIDs := make([]int64, 0)
				queryString := "update user_details set center_ids = $1 where user_id=$2"
				_, e := db.Exec(queryString, pq.Array(emptyCenterIDs), userID)
				if e != nil {
					fmt.Printf("DB Error - %v", e)
					fmt.Println()
				}
			}
			fmt.Println("Slots not available")
		}
		pollCallbackFunc(userID, callbackResponse)
	}

	currentSlot = (currentSlot + 1) % 3
}

// Poll is a func
func Poll(pollCallback func(userID string, response Response)) {
	pollCallbackFunc = pollCallback
	c := cron.New()
	c.AddFunc("@every 15s", pollAvailability)
	fmt.Println("Starting cron")
	c.Start()
}

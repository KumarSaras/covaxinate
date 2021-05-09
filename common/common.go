package common

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = ""
	password = ""
	dbname   = ""
)

// Register is func
func Register(user User) (Response, error) {

	// db := openDBConn()
	// defer db.Close()
	// stmt, err := db.Prepare("insert into user_details(user_id, district_id, slot) values( $1, $2, $3 )")
	// if err != nil {
	// 	return Response{}, fmt.Errorf("cannot prepare statement - %v", err)
	// }
	// _, err = stmt.Exec(user.ID, user.District, getSlot())
	// if err != nil {
	// 	return Response{}, fmt.Errorf("cannot insert user - %v", err)
	// }

	client := &http.Client{}
	reqURL := fmt.Sprintf("https://cdn-api.co-vin.in/api/v2/appointment/sessions/public/findByDistrict?district_id=%v&date=%v", strconv.Itoa(user.District), "10-05-2021")
	req, _ := http.NewRequest(http.MethodGet, reqURL, nil)
	req.Header.Add("Accept-Language", "hi_IN")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.76 Safari/537.36")

	res, resErr := client.Do(req)
	if resErr != nil {
		return Response{}, resErr
	}

	var jsonResponse Response

	json.NewDecoder(res.Body).Decode(&jsonResponse)

	if len(jsonResponse.Sessions) == 0 {
		return Response{}, fmt.Errorf("no vaccination center available")
	}

	return jsonResponse, nil
}

func getSlot() int {
	rand.Seed(time.Now().UnixNano())
	return (rand.Intn(3) + 1)
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

// func getAvailability(jsonResponse Response) {
// 	if len(jsonResponse.Sessions) {
// 		for _, session := range jsonResponse.Sessions {
// 		}
// 	}
// }

// Poll is a func
func Poll() {

}

package main

import (
	"fmt"

	"github.com/KumarSaras/covaxinate/common"
	"github.com/slack-go/slack"
)

func main() {
	slack.New("")
	user := common.User{
		ID:       "test",
		District: 257,
	}
	res, err := common.Register(user)
	if err != nil {
		panic(err)
	}

	fmt.Println(res.Sessions[0].CenterID)
}

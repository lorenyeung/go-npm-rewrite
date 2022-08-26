package main

import (
	"encoding/json"
	"fmt"

	"github.com/lorenyeung/go-npm-rewrite/auth"
	"github.com/lorenyeung/go-npm-rewrite/helpers"
	log "github.com/sirupsen/logrus"
)

type Aql struct {
	Result []Results `json:"results"`
}

type Results struct {
	Repo string `json:"repo"`
	Path string `json:"path"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func main() {
	//logic get aql results
	//for each path check if exists via head request
	//if doesnt exist, copy (add option for dry run)
	// allow filter for repo, scope (default *) etc

	flags := helpers.SetFlags()
	helpers.SetLogger(flags.LogLevelVar)

	stringFlags := map[string]string{"-user": flags.UsernameVar, "-apikey": flags.ApikeyVar, "-url": flags.URLVar}

	for i := range stringFlags {
		if stringFlags[i] == "" {
			log.Error(i + " cannot be empty")
		}
	}
	scope := "*"
	repo := "npm"
	aql := "items.find({\"repo\": {\"$match\":\"" + repo + "\"},\"path\":{\"$match\":\"@" + scope + "/*/@" + scope + "\"}}).include(\"repo\",\"name\",\"path\",\"type\")"

	log.Debug("AQL query created:" + aql)
	data, _, _, _ := auth.GetRestAPI("POST", true, flags.URLVar+"/artifactory/api/search/aql", flags.UsernameVar, flags.ApikeyVar, "", []byte(aql), map[string]string{"Content-Type": "text/plain"}, 0, flags, nil)

	var results Aql
	err := json.Unmarshal(data, &results)
	if err != nil {
		log.Error(err)
	}

	for result := range results.Result {
		fmt.Println(results.Result[result].Name)
	}
}

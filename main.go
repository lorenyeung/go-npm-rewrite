package main

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

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

	stringFlags := map[string]string{"-user": flags.UsernameVar, "-apikey": flags.ApikeyVar, "-url": flags.URLVar, "-repo": flags.RepoVar}

	for i := range stringFlags {
		if stringFlags[i] == "" {
			log.Error(i + " cannot be empty")
			os.Exit(1)
		}
	}
	scope := flags.ScopeVar
	if flags.ScopeVar == "" {
		scope = "*"
		log.Debug("custom scope:" + scope)
	}
	repo := flags.RepoVar
	aql := "items.find({\"repo\": {\"$match\":\"" + repo + "\"},\"path\":{\"$match\":\"@" + scope + "/*/@" + scope + "\"}}).include(\"repo\",\"name\",\"path\",\"type\")"

	log.Debug("AQL query created:" + aql)
	data, _, _, _ := auth.GetRestAPI("POST", true, flags.URLVar+"/artifactory/api/search/aql", flags.UsernameVar, flags.ApikeyVar, "", []byte(aql), map[string]string{"Content-Type": "text/plain"}, 0, flags, nil)

	var results Aql
	err := json.Unmarshal(data, &results)
	if err != nil {
		log.Error(err)
	}

	for result := range results.Result {
		if results.Result[result].Type == "file" {
			existingPath := results.Result[result].Repo + "/" + results.Result[result].Path + "/" + results.Result[result].Name
			paths := strings.Split(existingPath, "/")
			correctPath := paths[0] + "/" + paths[1] + "/" + paths[2] + "/-/" + results.Result[result].Name
			data, respCode, _, _ := auth.GetRestAPI("HEAD", true, flags.URLVar+"/artifactory/"+correctPath, flags.UsernameVar, flags.ApikeyVar, "", nil, nil, 0, flags, nil)
			log.Debug("checking if ", string(data), " exists via response code:", respCode, correctPath)
			if respCode == 404 {
				//attempt copy
				query := "/api/copy/" + existingPath + "?to=/" + correctPath + "&dry=" + strconv.Itoa(flags.DryRunVar)
				log.Debug(flags.URLVar + "/artifactory" + query)
				data2, respCode2, _, _ := auth.GetRestAPI("POST", true, flags.URLVar+"/artifactory"+query, flags.UsernameVar, flags.ApikeyVar, "", nil, nil, 0, flags, nil)
				log.Info(string(data2), respCode2)
			} else {
				log.Info(correctPath + " already exists, skipping")
			}
		}
	}
}

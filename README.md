# go-npm-rewrite

Required flags
*  -url string
    	Binary Manager URL without/artifactory (http://repo.com/)
*  -apikey string
    	API key or password
*  -repo string
    	Repository
*  -user string
      Username to run the script with

Optional flags
*  -scope string
    	Scope you want to of the packages you want to move. default is '*', otherwise you can set it
*  -log string
      Log level
      
      
Example run:
./rewrite-darwin-x64 -url http://my.artifactory:8082 -repo npm -user admin -apikey password -log debug -scope loren-dev

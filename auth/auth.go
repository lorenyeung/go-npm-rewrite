package auth

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/lorenyeung/go-npm-rewrite/helpers"
	log "github.com/sirupsen/logrus"
)

//Creds struct for creating download.json
type Creds struct {
	URL        string
	Username   string
	Apikey     string
	DlLocation string
}

//StorageDataJSON storage summary JSON
type StorageDataJSON struct {
	StorageSummary struct {
		FileStoreSummary struct {
			UsedSpace string `json:"usedSpace"`
			FreeSpace string `json:"freeSpace"`
		} `json:"fileStoreSummary"`
		RepositoriesSummaryList []struct {
			RepoKey string `json:"repoKey"`
		} `json: "repositoriesSummaryList"`
	} `json:"storageSummary"`
}

// VerifyAPIKey for errors
// func VerifyAPIKey(urlInput, userName, apiKey string) bool {
// 	log.Debug("starting VerifyAPIkey request. Testing:", userName)
// 	//TODO need to sanitize invalid url strings, esp in custom flag
// 	data, _, _, _ := GetRestAPI("GET", true, urlInput+"/api/system/ping", userName, apiKey, "", nil, nil, 1)
// 	if string(data) == "OK" {
// 		log.Debug("finished VerifyAPIkey request. Credentials are good to go.")
// 		return true
// 	}
// 	log.Warn("Received unexpected response:", string(data), " against ", urlInput+"/api/system/ping. Double check your URL and credentials.")
// 	return false
// }

// GenerateDownloadJSON (re)generate download JSON. Tested.
// func GenerateDownloadJSON(configPath string, regen bool, masterKey string) Creds {
// 	var creds Creds
// 	if regen {
// 		creds = GetDownloadJSON(configPath, masterKey)
// 	}
// 	var urlInput, userName, apiKey string
// 	reader := bufio.NewReader(os.Stdin)
// 	for {
// 		fmt.Printf("Enter your url [%s]: ", creds.URL)
// 		urlInput, _ = reader.ReadString('\n')
// 		urlInput = strings.TrimSuffix(urlInput, "\n")
// 		if urlInput == "" {
// 			urlInput = creds.URL
// 		}
// 		if !strings.HasPrefix(urlInput, "http") {
// 			fmt.Println("Please enter a HTTP(s) protocol")
// 			continue
// 		}
// 		if strings.HasSuffix(urlInput, "/") {
// 			log.Debug("stripping trailing /")
// 			urlInput = strings.TrimSuffix(urlInput, "/")
// 		}
// 		fmt.Printf("Enter your username [%s]: ", creds.Username)
// 		userName, _ = reader.ReadString('\n')
// 		userName = strings.TrimSuffix(userName, "\n")
// 		if userName == "" {
// 			userName = creds.Username
// 		}
// 		fmt.Print("Enter your API key/Password: ")
// 		apiKeyByte, _ := terminal.ReadPassword(0)
// 		apiKey = string(apiKeyByte)
// 		println()
// 		if VerifyAPIKey(urlInput, userName, apiKey) {
// 			break
// 		} else {
// 			fmt.Println("Something seems wrong, please try again.")
// 		}
// 	}
// 	dlLocationInput := configPath
// 	return writeFileDownloadJSON(configPath, urlInput, userName, apiKey, dlLocationInput, masterKey)
// }

func writeFileDownloadJSON(configPath, urlInput, userName, apiKey, dlLocationInput, masterKey string) Creds {
	data := Creds{
		URL:        Encrypt(urlInput, masterKey),
		Username:   Encrypt(userName, masterKey),
		Apikey:     Encrypt(apiKey, masterKey),
		DlLocation: Encrypt(dlLocationInput, masterKey),
	}
	//should probably encrypt data here
	fileData, err := json.Marshal(data)
	helpers.Check(err, true, "The JSON marshal", helpers.Trace())
	err2 := ioutil.WriteFile(configPath, fileData, 0600)
	helpers.Check(err2, true, "The JSON write", helpers.Trace())

	data2 := Creds{
		URL:        urlInput,
		Username:   userName,
		Apikey:     apiKey,
		DlLocation: dlLocationInput,
	}

	return data2
}

//GetDownloadJSON get data from DownloadJSON
// func GetDownloadJSON(fileLocation string, masterKey string) Creds {
// 	var result map[string]interface{}
// 	var resultData Creds
// 	file, err := os.Open(fileLocation)
// 	if err != nil {
// 		log.Error("error:", err)
// 		resultData = GenerateDownloadJSON(fileLocation, false, masterKey)
// 	} else {
// 		//should decrypt here
// 		defer file.Close()
// 		byteValue, _ := ioutil.ReadAll(file)
// 		json.Unmarshal([]byte(byteValue), &result)
// 		resultData.URL = Decrypt(result["URL"].(string), masterKey)
// 		resultData.Username = Decrypt(result["Username"].(string), masterKey)
// 		resultData.Apikey = Decrypt(result["Apikey"].(string), masterKey)
// 		resultData.DlLocation = Decrypt(result["DlLocation"].(string), masterKey)
// 	}
// 	return resultData
// }

//GetRestAPI GET rest APIs response with error handling
func GetRestAPI(method string, auth bool, urlInput, userName, apiKey, providedfilepath string, jsonBody []byte, header map[string]string, retry int, flags helpers.Flags, err error) ([]byte, int, http.Header, error) {
	if retry > flags.HTTPRetryMaxVar {
		log.Error("Exceeded retry limit, cancelling further attempts")
		return nil, 0, nil, err
	}

	body := new(bytes.Buffer)
	//PUT upload file
	if method == "PUT" && providedfilepath != "" {
		//req.Header.Set()
		file, err := os.Open(providedfilepath)
		helpers.Check(err, false, "open", helpers.Trace())
		defer file.Close()

		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", filepath.Base(providedfilepath))
		helpers.Check(err, false, "create", helpers.Trace())
		io.Copy(part, file)
		err = writer.Close()
		helpers.Check(err, false, "writer close", helpers.Trace())
	} else if (method == "PUT" || method == "POST") && jsonBody != nil {
		body = bytes.NewBuffer(jsonBody)
	}

	client := http.Client{}
	req, err := http.NewRequest(method, urlInput, body)

	//https://stackoverflow.com/questions/17714494/golang-http-request-results-in-eof-errors-when-making-multiple-requests-successi
	req.Close = true
	if auth {
		req.SetBasicAuth(userName, apiKey)
	}
	for x, y := range header {
		log.Debug("Recieved extra header:", x+":"+y)
		req.Header.Set(x, y)
	}

	if err != nil {
		log.Warn("The HTTP request failed with error", err)
	} else {

		resp, err := client.Do(req)
		if err != nil {
			log.Warn("The HTTP request failed with error:", err)
			time.Sleep(time.Duration(flags.HTTPSleepSecondsVar) * time.Second)
			GetRestAPI(method, auth, urlInput, userName, apiKey, providedfilepath, jsonBody, header, retry+1, flags, err)
		}
		// need to account for 403s with xray, or other 403s, 429? 204 is bad too (no content for docker)
		if resp == nil {
			log.Error("Returning error due to nil response on request:", err)
			return nil, 0, nil, err
		}
		switch resp.StatusCode {
		case 200:
			log.Debug("Received ", resp.StatusCode, " OK on ", method, " request for ", urlInput, " continuing")
		case 201:
			if method == "PUT" {
				log.Debug("Received ", resp.StatusCode, " ", method, " request for ", urlInput, " continuing")
			}
		case 403:
			log.Error("Received ", resp.StatusCode, " Forbidden on ", method, " request for ", urlInput, " continuing")
			// should we try retry here? probably not
		case 404:
			log.Debug("Received ", resp.StatusCode, " Not Found on ", method, " request for ", urlInput, " continuing")
		case 429:
			log.Error("Received ", resp.StatusCode, " Too Many Requests on ", method, " request for ", urlInput, ", sleeping then retrying, attempt ", retry)
			time.Sleep(time.Duration(flags.HTTPSleepSecondsVar) * time.Second)
			GetRestAPI(method, auth, urlInput, userName, apiKey, providedfilepath, jsonBody, header, retry+1, flags, err)
		case 204:
			if method == "GET" {
				log.Error("Received ", resp.StatusCode, " No Content on ", method, " request for ", urlInput, ", sleeping then retrying")
				time.Sleep(10 * time.Second)
				GetRestAPI(method, auth, urlInput, userName, apiKey, providedfilepath, jsonBody, header, retry+1, flags, err)
			} else {
				log.Debug("Received ", resp.StatusCode, " OK on ", method, " request for ", urlInput, " continuing")
			}
		case 500:
			log.Error("Received ", resp.StatusCode, " Internal Server error on ", method, " request for ", urlInput, " failing out")
			return nil, resp.StatusCode, nil, err
		case 502:
			log.Error("Received ", resp.StatusCode, " Internal Server error on ", method, " request for ", urlInput, " failing out")
			return nil, resp.StatusCode, nil, err
		case 503:
			log.Error("Received ", resp.StatusCode, " Internal Server error on ", method, " request for ", urlInput, " failing out")
			return nil, resp.StatusCode, nil, err
		default:
			log.Warn("Received ", resp.StatusCode, " on ", method, " request for ", urlInput, " continuing")
		}
		//Mostly for HEAD requests
		statusCode := resp.StatusCode
		headers := resp.Header

		if providedfilepath != "" && method == "GET" {
			// Create the file
			out, err := os.Create(providedfilepath)
			helpers.Check(err, false, "File create:"+providedfilepath, helpers.Trace())
			defer out.Close()

			//done := make(chan int64)
			//go helpers.PrintDownloadPercent(done, filepath, int64(resp.ContentLength))
			_, err = io.Copy(out, resp.Body)
			helpers.Check(err, false, "The file copy:"+providedfilepath, helpers.Trace())

			//return OK after copy is done
			return nil, 0, nil, nil
		} else {
			//maybe skip the download or retry if error here, like EOF
			data, err := ioutil.ReadAll(resp.Body)
			helpers.Check(err, false, "Data read:"+urlInput, helpers.Trace())
			if err != nil {
				log.Warn("Data Read on ", urlInput, " failed with:", err, ", sleeping then retrying, attempt:", retry)
				time.Sleep(time.Duration(flags.HTTPSleepSecondsVar) * time.Second)

				GetRestAPI(method, auth, urlInput, userName, apiKey, providedfilepath, jsonBody, header, retry+1, flags, err)
			}

			return data, statusCode, headers, nil
		}
	}
	return nil, 0, nil, err
}

//CreateHash self explanatory
func CreateHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

//Encrypt self explanatory
func Encrypt(dataString string, passphrase string) string {
	data := []byte(dataString)
	block, _ := aes.NewCipher([]byte(CreateHash(passphrase)))
	gcm, err := cipher.NewGCM(block)
	helpers.Check(err, true, "Cipher", helpers.Trace())
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.RawURLEncoding.EncodeToString([]byte(ciphertext))
}

//Decrypt self explanatory
func Decrypt(dataString string, passphrase string) string {
	data, _ := base64.RawURLEncoding.DecodeString(dataString)

	key := []byte(CreateHash(passphrase))
	block, err := aes.NewCipher(key)
	helpers.Check(err, true, "Cipher", helpers.Trace())
	gcm, err := cipher.NewGCM(block)
	helpers.Check(err, true, "Cipher GCM", helpers.Trace())
	// TODO if decrypt failure
	//	if err != nil {
	// 	GenerateDownloadJSON(fileLocation, false, passphrase)
	// }
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	helpers.Check(err, true, "GCM open", helpers.Trace())
	return string(plaintext)
}

//VerifyMasterKey self explanatory
func VerifyMasterKey(configPath string) string {
	_, err := os.Open(configPath)
	var token string
	if err != nil {
		log.Warn("Finding master key failed with error %s\n", err)
		data, err := generateRandomBytes(32)
		helpers.Check(err, true, "Generating new master key", helpers.Trace())
		err2 := ioutil.WriteFile(configPath, []byte(base64.URLEncoding.EncodeToString(data)), 0600)
		helpers.Check(err2, true, "Master key write", helpers.Trace())
		log.Info("Successfully generated master key")
		token = base64.URLEncoding.EncodeToString(data)
	} else {
		dat, err := ioutil.ReadFile(configPath)
		helpers.Check(err, true, "Reading master key", helpers.Trace())
		token = string(dat)
	}
	return token
}

func generateRandomString(s int) (string, error) {
	b, err := generateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}
	return b, nil
}

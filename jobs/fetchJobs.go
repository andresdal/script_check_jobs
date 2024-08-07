package jobs

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"neptune/check_jobs/slack_utils"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"github.com/go-redis/redis"
)

// Estructura para decodificar el JSON
type JobResult struct {
	URL string `json:"url"`
}

type WallaJobResponse struct {
	Jobs []JobResult `json:"jobs"`
}

type DecodedJobURL struct {
	URL string `json:"url"`
}


var keywords = []string{"driver", "developer", "designer", "nurse"}
var locations = []string{"New York, NY", "San Francisco, CA", "Chicago, IL", "Orlando, FL"}
var emails = []string{"test1@example.com", "test2@example.com", "test3@example.com", "walter@neptuneads.com"}

func ApiRequest(apiUrl string) ([]byte, string) {
	resp, err := http.Get(apiUrl)
	if err != nil {
		return nil, "Error in making the request: " + err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slack_utils.SendMessage("Error: Status code " + strconv.Itoa(resp.StatusCode), os.Getenv("CHANNEL_ID"))
		return nil, "Error: Status code " + strconv.Itoa(resp.StatusCode)
	}

	// Leer el cuerpo de la respuesta
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "Error in reading the request's body : " + err.Error()
	}

	return body, ""
}

func ConvertJsonHireable(body []byte) ([]JobResult, string) {
	var jobResults []JobResult
	err := json.Unmarshal(body, &jobResults)

	if err != nil {
		return nil, "Error decoding JSON: " + err.Error()
	}

	if len(jobResults) == 0 {
		return nil, "Error: No results found"
	}

	return jobResults, ""
}

func fetchRandomLink(jobResults []JobResult) (DecodedJobURL, string, JobResult) {
	randomJob := jobResults[rand.Intn(len(jobResults))]
	encodedPath := strings.Split(strings.Split(randomJob.URL, "job/")[1], "?")[0]
	decoded, err := base64.StdEncoding.DecodeString(encodedPath)

	if err != nil {
		return DecodedJobURL{}, "Error decoding the link: " + err.Error(), randomJob
	}

	var decodedJobURL DecodedJobURL
	err = json.Unmarshal(decoded, &decodedJobURL)

	if err != nil {
		return DecodedJobURL{}, "Error decoding the JSON's link: " + err.Error(), randomJob
	}

	return decodedJobURL, "", randomJob
}

func finalURLrequest(decodedJobURL DecodedJobURL) string {
	finalResp, err := http.Get(decodedJobURL.URL)
	if err != nil {
		return "Error following the final link: " + err.Error()
	}
	defer finalResp.Body.Close()

	// ignore error 403
	if finalResp.StatusCode == http.StatusForbidden {
		return ""
	}

	if finalResp.StatusCode != http.StatusOK {
		slack_utils.SendMessage("Error: Status code " + strconv.Itoa(finalResp.StatusCode), os.Getenv("CHANNEL_ID"))
		return "Error following the final link: Status code " + strconv.Itoa(finalResp.StatusCode)
	}

	return ""
}

func convertJsonWalla(body []byte) ([]JobResult, string) {
	var jobResponse WallaJobResponse
	err := json.Unmarshal(body, &jobResponse)

	if err != nil {
		return nil, "Error decoding JSON: " + err.Error()
	}

	jobResults := jobResponse.Jobs

	if len(jobResponse.Jobs) == 0 {
		return nil, "Error: No results found"
	}

	return jobResults, ""
}

func checkAPIHirable(apiUrl string, channelID string, redis_client *redis.Client) {
	gralErrorMessage := "FetchJobs Hireable error:"

	// Realizar la solicitud HTTP GET 
	body, errorMessage := ApiRequest(apiUrl)
	if body == nil{
		result, _ := redis_client.Get("check_api_hirable").Result()
		if result != "error" {
			redis_client.Set("check_api_hirable", "error", 0)
			slack_utils.SendMessage(gralErrorMessage + "\n" + errorMessage, channelID)
		}
		return
	}

	// Convertir el resultado a un objeto JSON
	jsonResults, errorMessage := ConvertJsonHireable(body)
	if jsonResults == nil {
		result, _ := redis_client.Get("check_api_hirable").Result()
		if result != "error" {
			redis_client.Set("check_api_hirable", "error", 0)
			slack_utils.SendMessage(gralErrorMessage + "\n" + errorMessage, channelID)
		}
		return
	}

	// Seleccionar un link aleatorio y seguirlo
	decodedJobURL, errorMessage, randomJob := fetchRandomLink(jsonResults)

	genericErrorMessage := "Unable to follow job link (Hirable): " + randomJob.URL

	if decodedJobURL.URL == "" {
		result, _ := redis_client.Get("check_api_hirable").Result()
		if result != "error" {
			redis_client.Set("check_api_hirable", "error", 0)
			slack_utils.SendMessage(genericErrorMessage + "\n" + errorMessage, channelID)
		}
		return
	}
	
	// Hacer una solicitud al URL final
	errorMessage = finalURLrequest(decodedJobURL)
	if errorMessage != "" {
		result, _ := redis_client.Get("check_api_hirable").Result()
		if result != "error" {
			redis_client.Set("check_api_hirable", "error", 0)
			slack_utils.SendMessage(genericErrorMessage + "\n" + errorMessage, channelID)
		}
		return
	}

	// if there is no error
	result, _ := redis_client.Get("check_api_hirable").Result()
	if result == "error" {
		redis_client.Set("check_api_hirable", "solved", 0)
		slack_utils.SendMessage("FetchJobs Hireable error SOLVED", channelID)
	}

	// LOG
	// slack_utils.SendMessage("(HIRABLE)" + "\n" + randomJob.URL + "\n", channelID)
	// slack_utils.SendMessage("Script ejecutado exitosamente. Se accedió al URL final: "+decodedJobURL.URL, channelID)
}

func checkAPIWalla(apiUrl string, channelID string, redis_client *redis.Client) {
	gralErrorMessage := "FetchJobs Walla error:"

	// Realizar la solicitud HTTP GET 
	body, errorMessage := ApiRequest(apiUrl)
	if body == nil{
		result, _ := redis_client.Get("check_api_walla").Result()
		if result != "error" {
			redis_client.Set("check_api_walla", "error", 0)
			slack_utils.SendMessage(gralErrorMessage + "\n" + errorMessage, channelID)
		}
		return
	}

	// Convertir el resultado a un objeto JSON
	jsonResults, errorMessage := convertJsonWalla(body)
	if jsonResults == nil {
		result, _ := redis_client.Get("check_api_walla").Result()
		if result != "error" {
			redis_client.Set("check_api_walla", "error", 0)
			slack_utils.SendMessage(gralErrorMessage + "\n" + errorMessage, channelID)
		}
		return
	}

	// Seleccionar un link aleatorio y seguirlo
	decodedJobURL, errorMessage, randomJob := fetchRandomLink(jsonResults)

	genericErrorMessage := "Unable to follow job link (Walla): " + randomJob.URL

	if decodedJobURL.URL == "" {
		result, _ := redis_client.Get("check_api_walla").Result()
		if result != "error" {
			redis_client.Set("check_api_walla", "error", 0)
			slack_utils.SendMessage(gralErrorMessage + "\n" + genericErrorMessage + "\n" + errorMessage, channelID)
		}
		return
	}

	// Hacer una solicitud al URL final
	errorMessage = finalURLrequest(decodedJobURL)
	if errorMessage != "" {
		result, _ := redis_client.Get("check_api_walla").Result()
		if result != "error" {
			redis_client.Set("check_api_walla", "error", 0)
			slack_utils.SendMessage(genericErrorMessage + "\n" + errorMessage, channelID)
		}
		return
	}

	// if there is no error
	result, _ := redis_client.Get("check_api_walla").Result()
	if result == "error" {
		redis_client.Set("check_api_walla", "solved", 0)
		slack_utils.SendMessage("FetchJobs Walla error SOLVED", channelID)
	}

	// LOG
	// slack_utils.SendMessage("(WALLA)" + "\n" + randomJob.URL + "\n", channelID)
	// slack_utils.SendMessage("Script ejecutado exitosamente. Se accedió al URL final: "+decodedJobURL.URL, channelID)
}

func FetchJobs(channelID string) {
	redis_client := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379", // Dirección del servidor Redis
        Password: "",               // Contraseña, si no tienes una, déjala vacía
        DB:       0,                // Base de datos a usar
    })

	keyword := keywords[rand.Intn(len(keywords))]
	location := locations[rand.Intn(len(locations))]
	email := emails[rand.Intn(len(emails))]

	var api_token1 = os.Getenv("API_TOKEN1")
	var api_token2 = os.Getenv("API_TOKEN2")

	apiUrlHirable := fmt.Sprintf("https://hireable.careerhotshot.com/search/?q=%s&l=%s&siteid=jobsclassic&jpp=150&email=%s&token=%s&from=sites&campaign=null&m_list_id=null",
		url.QueryEscape(keyword), url.QueryEscape(location), url.QueryEscape(email), api_token1)

	apiUrlWalla := fmt.Sprintf("https://walla.careerhotshot.com/search/?q=%s&l=%s&siteid=careerjobplacement&token=%s&jpp=15",
		url.QueryEscape(keyword), url.QueryEscape(location), api_token2)


	checkAPIHirable(apiUrlHirable, channelID, redis_client)
	checkAPIWalla(apiUrlWalla, channelID, redis_client)
}
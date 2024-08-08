package jobs

import (
	"neptune/check_jobs/slack_utils"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-redis/redis"
)


func check200(url string) string{
	finalResp, err := http.Get(url)

	if err != nil {
		return "Link: " + url + "\nError following the final link: " + err.Error()
	} 
	if finalResp == nil {
		return "Link: " + url + "\nError: final response is nil"
	} else if finalResp.StatusCode != http.StatusOK {
		defer finalResp.Body.Close()
		slack_utils.SendMessage("Error: Status code " +  strconv.Itoa(finalResp.StatusCode), os.Getenv("CHANNEL_ID"))
		return "Link: " + url + "\nError NOT OK: Status code " + strconv.Itoa(finalResp.StatusCode)
	} 
	return ""
}

func CheckEndpoint(channelID string) {
	redis_client := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379", // Dirección del servidor Redis
        Password: "",               // Contraseña, si no tienes una, déjala vacía
        DB:       0,                // Base de datos a usar
    })

	endpoint := "https://hireable.careerhotshot.com/api_feeds/sites"
	sites_exceptions := []string{"http://jobsandjobs.com", "https://searchprojobs.com", "http://search.topdirectjobs.com", "https://newjobfast.com/", "http://jobsandmore.com", "https://jobs.idropnews.com/"}
	errorMessageTitle := "Hireable endpoint errors:"

	// call API
	body, errorMessage := ApiRequest(endpoint)
	if body == nil{
		slack_utils.SendMessage(errorMessageTitle + "\n" + errorMessage, channelID)
	}

	// Convert result to JSON object
	jsonResults, errorMessage := ConvertJsonHireable(body)
	if jsonResults == nil {
		slack_utils.SendMessage(errorMessageTitle + "\n" + errorMessage, channelID)
	}

	var messages []string
	// go through each job and check if the request is 200
	for _, job := range jsonResults {
		// check if the job is in the exceptions list
		if contains(sites_exceptions, job.URL) {
			continue
		}
		var message = check200(job.URL)

		if message != "" {
			messages = append(messages, message) 
		}
	}
	
	result, _ := redis_client.Get("check_endpoints").Result()
	// if there are errors
	if len(messages) > 0 {
		if result != "error" {
			redis_client.Set("check_endpoints", "error", 0)
			slack_utils.SendMessage(errorMessageTitle + "\n" + strings.Join(messages, "\n"), channelID)
		}
	} else {
		if result == "error" {
			redis_client.Set("check_endpoints", "solved", 0)
			slack_utils.SendMessage("Endpoints error SOLVED", channelID)
		}
	}
}

func contains(s []string, str string) bool {
    for _, v := range s {
        if v == str {
            return true
        }
    }
    return false
}
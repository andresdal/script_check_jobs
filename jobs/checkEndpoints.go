package jobs

import (
	"neptune/check_jobs/slack_utils"
	"net/http"
	"strconv"
)


func check200(url string, channelID string) {
	finalResp, err := http.Get(url)

	if err != nil {
		slack_utils.SendMessage("Link: " + url + "\nError following the final link: " + err.Error(), channelID)
		println()
		return
	} 
	if finalResp == nil {
		slack_utils.SendMessage("Link: " + url + "\nError: final response is nil", channelID)
		println()
	} else if finalResp.StatusCode != http.StatusOK {
		defer finalResp.Body.Close()

		slack_utils.SendMessage("Link: " + url + "\nError NOT OK: Status code " + strconv.Itoa(finalResp.StatusCode), channelID)
		println()
	} 
}

func CheckEndpoint(channelID string) {
	endpoint := "https://hireable.careerhotshot.com/api_feeds/sites"
	
	// call API
	body, errorMessage := ApiRequest(endpoint)
	if body == nil{
		slack_utils.SendMessage(errorMessage, channelID)
	}

	// Convert result to JSON object
	jsonResults, errorMessage := ConvertJsonHireable(body)
	if jsonResults == nil {
		slack_utils.SendMessage(errorMessage, channelID)
	}

	// go through each job and check if the request is 200
	for _, job := range jsonResults {
		check200(job.URL, channelID)
	}
}
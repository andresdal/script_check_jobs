package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"neptune/check_jobs/slack_utils"
)

type OpenSearchConfig struct {
	Node     string
	Username string
	Password string
}

type Response struct {
    Took      int  `json:"took"`
    TimedOut  bool `json:"timed_out"`
    Shards    struct {
        Total      int `json:"total"`
        Successful int `json:"successful"`
        Skipped    int `json:"skipped"`
        Failed     int `json:"failed"`
    } `json:"_shards"`
    Hits struct {
        Total struct {
            Value    int    `json:"value"`
            Relation string `json:"relation"`
        } `json:"total"`
        MaxScore interface{} `json:"max_score"`
        Hits     []interface{} `json:"hits"`
    } `json:"hits"`
    Aggregations struct {
        GroupByCompany struct {
            DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
            SumOtherDocCount        int `json:"sum_other_doc_count"`
            Buckets                 []struct {
                Key      string `json:"key"`
                DocCount int    `json:"doc_count"`
            } `json:"buckets"`
        } `json:"group_by_company"`
    } `json:"aggregations"`
}

func CheckAmountFeeds(channelID string) {
	var sites = []Site{
		{"hireable", "https://hireable.careerhotshot.com/feeds-job-count"},
		{"walla", "https://jobsapi.jobsparser.com/feeds-job-count"},
	}

	neptuneConfig := OpenSearchConfig{
		Node:     os.Getenv("OPENSEARCH_NODE_NEPTUNE"),
		Username: os.Getenv("OPENSEARCH_USERNAME_NEPTUNE"),
		Password: os.Getenv("OPENSEARCH_PASSWORD_NEPTUNE"),
	}

	wallaConfig := OpenSearchConfig{
		Node:     os.Getenv("OPENSEARCH_NODE_WALLA"),
		Username: os.Getenv("OPENSEARCH_USERNAME_WALLA"),
		Password: os.Getenv("OPENSEARCH_PASSWORD_WALLA"),
	}

	for _, site := range sites {
		checkAmountFeedsSite(site, channelID, neptuneConfig, wallaConfig)
	}
}

func checkAmountFeedsSite(site Site, channelID string, neptuneConfig, wallaConfig OpenSearchConfig) {
	resp, err := http.Get(site.URL)
	if err != nil {
		slack_utils.SendMessage("Error making the request to the API: "+err.Error(), channelID)
		return
	}
	defer resp.Body.Close()

	// Decodificar la respuesta JSON
	var jobCounts []struct {
		FeedName string `json:"FeedName"`
		JobCount int    `json:"JobCount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jobCounts); err != nil {
		slack_utils.SendMessage("Error decoding the JSON response: "+err.Error(), channelID)
		return
	}

	// Realizar la consulta a OpenSearch para cada feed
	var queryError []string

	if site.Name == "hireable" {
		//slack_utils.SendMessage("Checking amount feeds for Hireable", channelID)
		for _, jc := range jobCounts {
			queryError = append(queryError, queryOpenSearch(neptuneConfig, jc.FeedName)) 
		}
	} else if site.Name == "walla" {
		//slack_utils.SendMessage("Checking amount feeds for Walla", channelID)
		for _, jc := range jobCounts {
			queryError = append(queryError, queryOpenSearch(wallaConfig, jc.FeedName)) 
		}
	}

	var gralErrorMessage = "CheckAmountFeeds " + site.Name + " error:"
	var errorMessage string
	for _, err := range queryError {
		if err != "OK" {
			errorMessage += err + "\n"
		}
	}

	if(errorMessage != "") {
		slack_utils.SendMessage(gralErrorMessage+"\n"+errorMessage, channelID)
	}
	
}

func queryOpenSearch(config OpenSearchConfig, feedProvider string) string {
	url := fmt.Sprintf("%s/_search", config.Node)
	query := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"feed_provider.keyword": feedProvider,
						},
					},
				},
			},
		},
		"aggs": map[string]interface{}{
			"group_by_company": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "feed_version.keyword",
					"size":  100,
				},
			},
		},
	}

	queryBytes, err := json.Marshal(query)
	if err != nil {
		return fmt.Sprintf("Error marshaling query for %s: %v", feedProvider, err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(queryBytes))
	if err != nil {
		return fmt.Sprintf("Error creating request for %s: %v", feedProvider, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(config.Username, config.Password)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Error making request to %s: %v", feedProvider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slack_utils.SendMessage("Error: Status code " + resp.Status, os.Getenv("CHANNEL_ID"))
		return fmt.Sprintf("Non-OK HTTP status: %s for %s", resp.Status, feedProvider)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error reading response body for %s: %v", feedProvider, err)
	}

	var result Response
    err = json.Unmarshal(body, &result)
    if err != nil {
        return fmt.Sprintf("Error unmarshalling response body for %s: %v", feedProvider, err)
    }

    if len(result.Aggregations.GroupByCompany.Buckets) > 1 {
		buckets, _ := json.Marshal(result.Aggregations.GroupByCompany.Buckets)
		return fmt.Sprintf("Error: more than one element in buckets for %s\nBuckets: %s ", feedProvider, string(buckets))
    }

	return "OK"
	//fmt.Sprintf("Response from %s:\n%s\n", feedProvider, body)
}

package jobs

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"neptune/check_jobs/slack_utils"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

type JobDetails struct {
    URL     string
    Source  string
}

func finalURLrequest2(url string) string {
	finalResp, err := http.Get(url)
	if err != nil {
		return "Error following the final link: " + err.Error()
	}
	defer finalResp.Body.Close()

	// ignore error 403
	if finalResp.StatusCode == http.StatusForbidden {
		return ""
	}

	if finalResp.StatusCode != http.StatusOK {
		return "Error following the final link: Status code " + strconv.Itoa(finalResp.StatusCode)
	}

	return ""
}

func followRedirects(urlStr string) {
    client := &http.Client{
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            if len(via) >= 10 {
                return fmt.Errorf("stopped after 10 redirects")
            }
            fmt.Printf("Redirect: %s\n", req.URL)
            return nil
        },
    }

    resp, err := client.Get(urlStr)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    defer resp.Body.Close()

    fmt.Printf("Final URL: %s\n", resp.Request.URL)
    fmt.Printf("Status Code: %d\n", resp.StatusCode)
}

func followRedirectsProxy(urlStr string) {
    proxyStr := "http://35.185.196.38:3128"
    proxyURL, err := url.Parse(proxyStr)
    if err != nil {
        fmt.Printf("Error parsing proxy URL: %v\n", err)
        return
    }

    client := &http.Client{
        Transport: &http.Transport{
            Proxy: http.ProxyURL(proxyURL),
        },
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            if len(via) >= 10 {
                return fmt.Errorf("stopped after 10 redirects")
            }
            fmt.Printf("Redirect: %s\n", req.URL)
            return nil
        },
    }

    resp, err := client.Get(urlStr)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    defer resp.Body.Close()

    fmt.Printf("Final URL: %s\n", resp.Request.URL)
    fmt.Printf("Status Code: %d\n", resp.StatusCode)
}

func CheckDiffSource(channelID string) {
	redis_client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Dirección del servidor Redis
		Password: "",               // Contraseña, si no tienes una, déjala vacía
		DB:       0,                // Base de datos a usar
	})

	var keywords = []string{"driver", "developer", "designer", "nurse"}
	var locations = []string{"New York, NY", "San Francisco, CA", "Chicago, IL", "Orlando, FL"}
	var emails = []string{"test1@example.com", "test2@example.com", "test3@example.com", "walter@neptuneads.com"}

	keyword := keywords[rand.Intn(len(keywords))]
	location := locations[rand.Intn(len(locations))]
	email := emails[rand.Intn(len(emails))]

	var api_token1 = os.Getenv("API_TOKEN1")

	apiUrlHirable := fmt.Sprintf("https://hireable.careerhotshot.com/search/?q=%s&l=%s&siteid=jobsclassic&jpp=150&email=%s&token=%s&from=sites&campaign=null&m_list_id=null",
		url.QueryEscape(keyword), url.QueryEscape(location), url.QueryEscape(email), api_token1)

	gralErrorMessage := "CheckDiffSource error:"

	// Realizar la solicitud HTTP GET 
	body, errorMessage := ApiRequest(apiUrlHirable)
	if body == nil{
		result, _ := redis_client.Get("check_diff_source").Result()
		if result != "error" {
			redis_client.Set("check_diff_source", "error", 0)
			slack_utils.SendMessage(gralErrorMessage + "\n" + errorMessage, channelID)
		}
		return
	}

	// Convertir el resultado a un objeto JSON
	jsonResults, errorMessage := ConvertJsonHireable(body)
	if jsonResults == nil {
		result, _ := redis_client.Get("check_diff_source").Result()
		if result != "error" {
			redis_client.Set("check_diff_source", "error", 0)
			slack_utils.SendMessage(gralErrorMessage + "\n" + errorMessage, channelID)
		}
		return
	}

	// Select two links with the same source 
	sourceMap := make(map[string][]JobDetails)

    // Decode job results and organize by source
    for _, job := range jsonResults {
        var decodedJob struct {
            URL    string `json:"url"`
            Source string `json:"source"`
        }
        encodedPath := strings.Split(strings.Split(job.URL, "job/")[1], "?")[0]
        decoded, err := base64.StdEncoding.DecodeString(encodedPath)
        if err != nil {
            continue // Skip jobs that can't be decoded
        }
        err = json.Unmarshal(decoded, &decodedJob)
        if err != nil {
            continue // Skip jobs that can't be unmarshaled
        }
        sourceMap[decodedJob.Source] = append(sourceMap[decodedJob.Source], JobDetails{URL: job.URL, Source: decodedJob.Source})
    }

	jobsList := []JobDetails{}
    // Find the first source with at least two jobs
    for _, jobs := range sourceMap {
        if len(jobs) >= 2 {
            jobsList = []JobDetails{jobs[0], jobs[1]}
        }
    }

    // if no two links with the same source found, show error
	if len(jobsList) == 0 {
		result, _ := redis_client.Get("check_diff_source").Result()
		if result != "error" {
			redis_client.Set("check_diff_source", "error", 0)
			slack_utils.SendMessage(gralErrorMessage + "\n no two links with the same source found", channelID)
		}
		return
	}

	fmt.Print(jobsList)
	print("\n")
	
	// Follow the first link
	followRedirects(jobsList[0].URL)

	time.Sleep(5 * time.Second)

	// Check if source is the same
	//followRedirects(jobsList[1].URL)

	// if there is no error
	// result, _ := redis_client.Get("check_diff_source").Result()
	// if result == "error" {
	// 	redis_client.Set("check_diff_source", "solved", 0)
	// 	slack_utils.SendMessage("CheckDiffSource error SOLVED", channelID)
	// }
}
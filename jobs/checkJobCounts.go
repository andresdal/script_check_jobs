package jobs

import (
	"encoding/json"
	"neptune/check_jobs/slack_utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

type Site struct {
    Name string
    URL  string
}

func CheckJobsCounts(c *cache.Cache, channelID string){
	var sites = []Site{ Site{"hireable", "https://hireable.careerhotshot.com/feeds-job-count"}, Site{"walla", "https://jobsapi.jobsparser.com/feeds-job-count"}}

	for _, site := range sites {
		checkJobCounts(c, channelID, site)
	}
}

func checkJobCounts(c *cache.Cache, channelID string, site Site) {
	resp, err := http.Get(site.URL)
	if err != nil {
		slack_utils.SendMessage("Error making the request to the API: " + err.Error(), channelID)
	}
	defer resp.Body.Close()

	// Decodificar la respuesta JSON
	var jobCounts []struct {
		FeedName string `json:"FeedName"`
		JobCount int    `json:"JobCount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jobCounts); err != nil {
		slack_utils.SendMessage("Error decoding the JSON response: " + err.Error(), channelID)
	}

	// Leer los números de trabajo anteriores desde la caché
	prevJobCounts, found := c.Get(site.URL)
	if !found {
		// Si no se encontraron los números anteriores, inicializar como un mapa vacío
		prevJobCounts = make(map[string]int)
	}

	// Convertir a tipo adecuado
	prevCounts := prevJobCounts.(map[string]int)

	// Guardar los números de trabajo actuales en la caché
	currentCounts := make(map[string]int)
	for _, jc := range jobCounts {
		currentCounts[jc.FeedName] = jc.JobCount
	}
	c.Set(site.URL, currentCounts, 24*time.Hour)

	// Comparar los números actuales con los anteriores y enviar una notificación si no cambian
	var messages []string
	for _, jc := range jobCounts {
		prevCount, ok := prevCounts[jc.FeedName]
		if !ok || prevCount == jc.JobCount {
			// Aquí puedes enviar una notificación
			messages = append(messages, "The JobCount of " + jc.FeedName + " has not changed. It is still " + strconv.Itoa(jc.JobCount))
		}
	}
	if len(messages) > 0 {
		slack_utils.SendMessage(site.Name + " jobCount errors:\n" + strings.Join(messages, "\n"), channelID)
	}
}
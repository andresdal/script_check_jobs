package jobs

import (
	"encoding/json"
	"neptune/check_jobs/slack_utils"
	"net/http"
	"strconv"
	"time"

	"github.com/patrickmn/go-cache"
)

func CheckJobCounts(c *cache.Cache, channelID string) {
	resp, err := http.Get("https://jobsapi.jobsparser.com/feeds-job-count")
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
	prevJobCounts, found := c.Get("jobCounts")
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
	c.Set("jobCounts", currentCounts, 24*time.Hour)

	// Comparar los números actuales con los anteriores y enviar una notificación si no cambian
	for _, jc := range jobCounts {
		prevCount, ok := prevCounts[jc.FeedName]
		if !ok || prevCount == jc.JobCount {
			// Aquí puedes enviar una notificación
			slack_utils.SendMessage("The JobCount of " + jc.FeedName + " has not changed. It is still " + strconv.Itoa(jc.JobCount), channelID)
		}
	}
}
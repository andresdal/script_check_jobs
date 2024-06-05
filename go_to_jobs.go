package main

import (
	"bufio"
	"fmt"
	"neptune/check_jobs/jobs"
	"os"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

var c *cache.Cache

func workerFetchJobs(interval time.Duration, channelID string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		jobs.FetchJobs(channelID)
	}
}

func workerCheckJobsCount(interval time.Duration, c *cache.Cache, channelID string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		jobs.CheckJobCounts(c, channelID)
	}
}

func workerCheckEndpoints(interval time.Duration, channelID string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		jobs.CheckEndpoint(channelID)
	}
}

func main() {
	loadEnv()
	c = cache.New(24*time.Hour, 1*time.Hour)
	channelID := os.Getenv("CHANNEL_ID")
	if channelID == "" {
		fmt.Println("SLACK_TOKEN environment variable not set")
		return
	}
	
	// go workerFetchJobs(5 * time.Minute, channelID)
	// go workerCheckJobsCount(12 * time.Hour, c, channelID)
	// go workerCheckEndpoints(1 * time.Hour, channelID)

	// // Mantener el programa en ejecución indefinidamente
	// select {}

	jobs.FetchJobs(channelID)
}

func loadEnv() {
    file, err := os.Open(".env")
    if err != nil {
        panic(err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        parts := strings.SplitN(scanner.Text(), "=", 2)
        if len(parts) != 2 {
            continue
        }
        key, value := parts[0], parts[1]
        value = strings.Trim(value, "\"") // remove quotes if present
        os.Setenv(key, value)
		fmt.Printf("Set %s=%s\n", key, value)
    }

    if err := scanner.Err(); err != nil {
        panic(err)
    }
}

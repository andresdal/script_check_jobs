package slack_utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// Struct para representar el cuerpo de la solicitud
type SlackMessage struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func SendSlackMessage(messageText string, channelID string) {
	token := os.Getenv("SLACK_TOKEN")
	if token == "" {
		fmt.Println("SLACK_TOKEN environment variable not set")
		return
	}

	message := SlackMessage{
		Channel: channelID,
		Text:    messageText,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Error al codificar el mensaje:", err)
		return
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(messageBytes))
	if err != nil {
		fmt.Println("Error al crear la solicitud:", err)
		return
	}

	// Agregar headers
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error al enviar la solicitud:", err)
		return
	}
	defer resp.Body.Close()

	// Leer y mostrar la respuesta
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	//fmt.Println("Respuesta de Slack:", result)
}

func SendMessage(errorMessage string, channelID string) {
	fmt.Println(errorMessage)
	SendSlackMessage(errorMessage, channelID)
}

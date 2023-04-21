package gpt

import (
	"bytes"
    "bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"github.com/qingconglaixueit/wechatbot/config"
)

// ChatGPTResponseBody 响应体
type ChatGPTResponseBody struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int                    `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChoiceItem           `json:"choices"`
	Usage   map[string]interface{} `json:"usage"`
	Error   struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Param   interface{} `json:"param"`
		Code    interface{} `json:"code"`
	} `json:"error"`
}

type Choice struct {
    Text string `json:"text"`
}

type Event struct {
    Choices []Choice `json:"choices"`
}


type ChoiceItem struct {
	Message      Message 	    `json:"message"`
	Index        int    		`json:"index"`
	FinishReason string 		`json:"finish_reason"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

// ChatGPTRequestBody 请求体
type ChatGPTRequestBody struct {
	Model            string  `json:"model"`
	MaxTokens        uint    `json:"max_tokens"`
	Temperature      float64 `json:"temperature"`
	TopP             int     `json:"top_p"`
	FrequencyPenalty int     `json:"frequency_penalty"`
	PresencePenalty  int     `json:"presence_penalty"`
    Stream            bool    `json:"stream"`
    Messages         []Message         `json:"messages"`
}



// Completions gtp文本模型回复
//curl https://api.openai.com/v1/completions
//-H "Content-Type: application/json"
//-H "Authorization: Bearer your chatGPT key"
//-d '{"model": "text-davinci-003", "prompt": "give me good song", "temperature": 0, "max_tokens": 7}'

func Completions(msg string) (string, error) {
	var gptResponseBody *ChatGPTResponseBody
	var resErr error
    start := time.Now()
    var reply string
	for retry := 1; retry <= 3; retry++ {
		if retry > 1 {
			time.Sleep(time.Duration(retry-1) * 100 * time.Millisecond)
		}
        
        reply, resErr = httpRequestCompletions(msg, retry)
		if resErr != nil {
			log.Printf("gpt request(%d) error: %v\n", retry, resErr)
			continue
		}
		if gptResponseBody.Error.Message == "" {
			break
		}
	}
	if resErr != nil {
		return "", resErr
	}
    elapsed := time.Since(start)
    log.Printf("API response time: %s\n", elapsed)
	return reply, nil
}

func httpRequestCompletions(msg string, runtimes int) (string, error) {
	cfg := config.LoadConfig()
	if cfg.ApiKey == "" {
		return "", errors.New("api key required")
	}
    startTime := time.Now()
    requestBody := ChatGPTRequestBody{
        Model:            cfg.Model,
        MaxTokens:        cfg.MaxTokens,
        Temperature:      cfg.Temperature,
        TopP:             1,
        FrequencyPenalty: 0,
        PresencePenalty:  0,
        Stream:           true,
        Messages:        []Message{
            {
                Role:    "system",
                Content: "You are a helpful assistant.",
            },
            {
                Role:    "user",
                Content: msg,
            },
        },
    }
    
    
	requestData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("json.Marshal requestBody error: %v", err)
	}
	
    log.Printf("gpt request(%d) json: %s\n", runtimes, string(requestData))
	
    req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestData))
	if err != nil {
		return "", fmt.Errorf("http.NewRequest error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ApiKey)
    
    client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("client.Do error: %v", err)
	}
    // Close the response body
	defer response.Body.Close()
    
// create variables to collect the stream of events
    var collectedEvents []Event
    var completionText string

    // read the response stream using a scanner
    scanner := bufio.NewScanner(response.Body)
    
    for scanner.Scan() {
        // decode the event from JSON
        var event Event
        err := json.Unmarshal(scanner.Bytes(), &event)
        if err != nil {
            return "", fmt.Errorf("client.Do error: %v", err)
        }

        // calculate the time delay of the event
        eventTime := time.Since(startTime).Seconds()

        // save the event response
        collectedEvents = append(collectedEvents, event)

        // extract the text and append to the completion text
        eventText := event.Choices[0].Text
        completionText += eventText

        // print the delay and text
        fmt.Printf("Text received: %s (%.2f seconds after request)\n", eventText, eventTime)
    }

    // print the time delay and text received
    fullTime := time.Since(startTime).Seconds()
    fmt.Printf("Full response received %.2f seconds after request\n", fullTime)
    fmt.Printf("Full text received: %s\n", completionText)
    
	return completionText, nil
}

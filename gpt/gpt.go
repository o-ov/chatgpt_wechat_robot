package gpt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
    "io"
	"net/http"
	"time"
    "bufio"
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

type StreamRes struct {
    Data *CreateCompletionStreamingResponse `json:"data"`
}

type CreateCompletionStreamingResponse struct {
    ID        string             `json:"id,omitempty"`
    Object    string             `json:"object,omitempty"`
    CreatedAt int64              `json:"created_at,omitempty"`
    Choices   []*StreamingChoice `json:"choices,omitempty"`
}

type StreamingChoice struct {
    Delta        *Message `json:"delta,omitempty"`
    Index        int      `json:"index,omitempty"`
    LogProbs     int      `json:"logprobs,omitempty"`
    FinishReason string   `json:"finish_reason,omitempty"`
}

type ChoiceItem struct {
	Message      Message 	    `json:"message"`
	Index        int    		`json:"index"`
	FinishReason string 		`json:"finish_reason"`
}

type Message struct {
    Role    string `json:"role,omitempty"`
    Content string `json:"content"`
    Name    string `json:"name,omitempty"`
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
	var resErr error
    start := time.Now()
    var reply string
	reply, resErr = httpStreamRequestCompletions(msg, 1)
	if resErr != nil {
		return "", resErr
	}
    elapsed := time.Since(start)
    log.Printf("API response time: %s\n", elapsed)
	return reply, nil
}

func httpStreamRequestCompletions(msg string, runtimes int) (string, error) {
    cfg := config.LoadConfig()
    if cfg.ApiKey == "" {
        return "", errors.New("api key required")
    }
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
    
    collectedMessages := make([]string, 0)

    // Create a new buffered reader for the response body
    reader := bufio.NewReader(response.Body)

    // Loop through each line in the response
    i := 1
    for {
        fmt.Println("for Loop %d",i)
        i++
        // Read a line from the response
        line, err := reader.ReadBytes('\n')
        
        if err != nil {
            if err == io.EOF {
                break
            }
            return "", fmt.Errorf("ReadBytes error: %v", err)
        }

        
        // Check if the line is the end of the stream
        // Remove the newline character from the line
        fmt.Println("Received JSON data:", string(line))
        if len(line)>6 {
            line = line[6:len(line)-1]
        if string(line) == "[DONE]" {
            fmt.Println("Stream finished")
            break
        }
            // Otherwise, assume the line is JSON data
            var collectedChunks CreateCompletionStreamingResponse
            err = json.Unmarshal(line, &collectedChunks)
            if err != nil {
                return "", fmt.Errorf("Unmarshal error: %v", err)
            }
            fmt.Printf("collectedChunks: %+v\n", collectedChunks)

            if collectedChunks.Choices != nil && len(collectedChunks.Choices) > 0  {
                // Content字段为空
                temp := collectedChunks.Choices[0]
                fmt.Printf("temp: %+v\n", temp)
                if temp.Delta != nil {
                    tmp := temp.Delta
                    fmt.Printf("tmp: %+v\n", tmp)
                    if tmp.Content != ""{
                        chunkMessage := tmp.Content // extract the message
                        fmt.Println("no 202" + chunkMessage)
                        collectedMessages = append(collectedMessages, chunkMessage) // save the message
                    }
                }
            }
        }
    }

    // print the time delay and text received
    fullReplyContent := ""
    for _, message := range collectedMessages {
        fullReplyContent += message
    }
    fmt.Printf("Full conversation received: %s\n", fullReplyContent)
    return fullReplyContent, nil
}


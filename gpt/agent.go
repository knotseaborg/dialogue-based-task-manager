package gpt

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type Spake struct {
	Source  string
	Content string
}

type APIError struct{}

func (e APIError) Error() string {
	return "Fatal error. API does not exist"
}

var agentPrompt string

func SpakeHandler(toAudioComm chan string, dialogue *Spake) error {
	/*Handles dialogues*/
	c := openai.NewClient(os.Getenv("DBTM_OPEN_AI_KEY"))
	if agentPrompt == "" {
		rawIntroduction, err := os.ReadFile("./gpt/prompt.txt")
		if err != nil {
			return nil
		}
		agentPrompt = string(rawIntroduction)
	}
	// add spake to prompt
	agentPrompt = fmt.Sprintf("%s\n%s: %s", agentPrompt, dialogue.Source, dialogue.Content)

	// emma speaks
	dialogue, err := emmaSpake(c)
	if err != nil {
		return err
	}

	if message, _, ok := strings.Cut(dialogue.Content, "::"); ok {
		toAudioComm <- message
	} else {
		toAudioComm <- dialogue.Content
	}

	// add emma's spake to prompt
	agentPrompt = fmt.Sprintf("%s\n%s: %s", agentPrompt, dialogue.Source, dialogue.Content)

	// get api response if there is one
	APIResponse, err := APIHandler(dialogue.Content)
	if err != nil {
		return err
	}
	if APIResponse != "" {
		// recursive call to get emma's response from api
		err := SpakeHandler(toAudioComm, &Spake{Source: "API_RESPONSE", Content: APIResponse})
		if err != nil {
			return err
		}
		return nil
	}
	fmt.Println(agentPrompt)
	return nil
}

func emmaSpake(c *openai.Client) (*Spake, error) {
	ctx := context.Background()
	req := openai.CompletionRequest{
		Model:       openai.GPT3TextDavinci003,
		MaxTokens:   500,
		Prompt:      agentPrompt + "\n" + "Emma:",
		Stop:        []string{"User:", "Emma:", "API_RESPONSE", "API Handler"},
		Temperature: 0.5,
		TopP:        0.2,
	}
	resp, err := c.CreateCompletion(ctx, req)
	if err != nil {
		return nil, err
	}
	return &Spake{Source: "Emma", Content: strings.Trim(resp.Choices[0].Text, "\n ")}, nil
}

func APIHandler(agentResponse string) (string, error) {

	directiveReg, _ := regexp.Compile(`::[A-Z_]+(/\d+)?`)
	dataReg, _ := regexp.Compile(`(?s){.+}`)
	directive := directiveReg.FindString(agentResponse)
	data := dataReg.FindString(agentResponse)
	fmt.Println("Directive detected: ", directive)
	if directive != "" {
		fmt.Println("Directive detected: ", directive)
		fmt.Println("Data detected: ", data)
		if command, activityID, ok := strings.Cut(directive, "/"); ok {
			switch command {
			default:
				return "", APIError{}
			case "::CREATE_FOLLOWUP_ACTIVITY":
				response, err := makeRequest("http://localhost:8080/followup/create/"+activityID, "PUT", data)
				if err != nil {
					return "", err
				}
				return response, nil
			case "::FOLLOWUP":
				response, err := makeRequest("http://localhost:8080/followup/"+activityID, "GET", "{}")
				if err != nil {
					return "", err
				}
				return response, nil
			case "::DELETE_ACTIVITY":
				response, err := makeRequest("http://localhost:8080/activity/"+activityID, "DELETE", "{}")
				if err != nil {
					return "", err
				}
				return response, nil
			}
		}
		switch directive {
		default:
			return "", APIError{}
		case "::CREATE_ACTIVITY":
			response, err := makeRequest("http://localhost:8080/activity/create", "PUT", data)
			if err != nil {
				return "", err
			}
			return response, nil
		case "::GET_ACTIVITIES":
			response, err := makeRequest("http://localhost:8080/activity", "POST", data)
			if err != nil {
				return "", err
			}
			return response, nil
		case "::EDIT_ACTIVITY":
			response, err := makeRequest("http://localhost:8080/activity/edit", "PUT", data)
			if err != nil {
				return "", err
			}
			return response, nil
		}
	}

	return "", nil
}

func makeRequest(uri string, requestType string, data string) (string, error) {
	//var responseBody APIResponse

	if data == "" {
		data = "{}"
	}
	client := &http.Client{}
	var ioData = strings.NewReader(data)
	req, err := http.NewRequest(requestType, uri, ioData)
	if err != nil {
		return "", nil
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil
	}

	fmt.Print("This is the raw return of API", string(body))

	// if err = json.Unmarshal(body, &responseBody); err != nil {
	// 	return nil, err
	// }

	return string(body), nil
}

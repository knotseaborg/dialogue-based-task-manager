package gpt

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/knotseaborg/dbtm/activity"
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

var agentInstruction string
var promptQueue []string

func SpakeHandler(toAudioComm chan string, dialogue *Spake) error {
	/*Handles dialogues*/
	c := openai.NewClient(os.Getenv("DBTM_OPEN_AI_KEY"))
	if agentInstruction == "" {
		rawIntroduction, err := os.ReadFile(os.Getenv("DBTM_PROMPT_PATH"))
		if err != nil {
			return nil
		}
		agentInstruction = string(rawIntroduction)
		agentInstruction = strings.Replace(agentInstruction, "!!CURRENT_TIME!!", time.Now().Format(activity.TIMEFORMAT), 1)
		agentInstruction = strings.Replace(agentInstruction, "!!USER_NAME!!", "Reuben", 1)
	}
	// add spake to prompt
	promptQueue = append(promptQueue, fmt.Sprintf("%s: %s", dialogue.Source, dialogue.Content))

	// emma speaks
	dialogue, err := emmaSpake(c)
	// expect an error, when there is a context overflow
	for retryLimit := 10; err != nil && retryLimit > 0; dialogue, err = emmaSpake(c) {
		// Pop message off queue and try again
		promptQueue = promptQueue[1:]
		retryLimit--
		log.Print("Token limit exceeded. Retrying ", retryLimit)
		log.Print(err)
	}

	if message, _, ok := strings.Cut(dialogue.Content, "::"); ok {
		toAudioComm <- message
	} else {
		toAudioComm <- dialogue.Content
	}

	// add emma's spake to prompt
	promptQueue = append(promptQueue, fmt.Sprintf("%s: %s", dialogue.Source, dialogue.Content))

	// get api response if there is one
	APIResponse, err := apiHandler(dialogue.Content)
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
	fmt.Println(strings.Join(promptQueue, "\n"))
	return nil
}

func emmaSpake(c *openai.Client) (*Spake, error) {

	prompt := fmt.Sprintf("%s%s", agentInstruction, strings.Join(promptQueue, "\n"))
	ctx := context.Background()
	req := openai.CompletionRequest{
		Model:       openai.GPT3TextDavinci003,
		MaxTokens:   500,
		Prompt:      fmt.Sprintf("%s\n%s", prompt, "Emma:"),
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

func apiHandler(agentResponse string) (string, error) {

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
				response, err := makeRequest(os.Getenv("DBTM_SERVER_URI")+"/followup/create/"+activityID, "PUT", data)
				if err != nil {
					return "", err
				}
				return response, nil
			case "::FOLLOWUP":
				response, err := makeRequest(os.Getenv("DBTM_SERVER_URI")+"/followup/"+activityID, "GET", "{}")
				if err != nil {
					return "", err
				}
				return response, nil
			case "::DELETE_ACTIVITY":
				response, err := makeRequest(os.Getenv("DBTM_SERVER_URI")+"/activity/"+activityID, "DELETE", "{}")
				if err != nil {
					return "", err
				}
				return response, nil
			case "::GET_ACTIVITY":
				response, err := makeRequest(os.Getenv("DBTM_SERVER_URI")+"/activity/"+activityID, "GET", "{}")
				if err != nil {
					return "", err
				}
				return response, nil
			}
		}
		switch directive {
		default:
			return "", APIError{}
		case "::TIME_NOW":
			response, err := makeRequest(os.Getenv("DBTM_SERVER_URI")+"/time", "GET", "{}")
			if err != nil {
				return "", err
			}
			return response, nil
		case "::CREATE_ACTIVITY":
			response, err := makeRequest(os.Getenv("DBTM_SERVER_URI")+"/activity/create", "PUT", data)
			if err != nil {
				return "", err
			}
			return response, nil
		case "::GET_ACTIVITIES":
			response, err := makeRequest(os.Getenv("DBTM_SERVER_URI")+"/activity", "POST", data)
			if err != nil {
				return "", err
			}
			return response, nil
		case "::EDIT_ACTIVITY":
			response, err := makeRequest(os.Getenv("DBTM_SERVER_URI")+"/activity/edit", "PUT", data)
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

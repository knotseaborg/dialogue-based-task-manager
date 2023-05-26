package voice

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type speechResponse struct {
	AudioContent string `json:"audioContent"`
}

func ToAudio(text string) (string, error) {
	//var responseBody APIResponse

	data := fmt.Sprintf(`{
		"input": {
		  "text": "%s"
		},
		"voice": {
		  "languageCode": "en-gb",
		  "name": "en-GB-Neural2-C",
		  "ssmlGender": "FEMALE"
		},
		"audioConfig": {
		  "audioEncoding": "MP3"
		}
	}`, text)

	client := &http.Client{}
	var ioData = strings.NewReader(data)
	req, err := http.NewRequest("POST", os.Getenv("DBTM_SPEECH_ENDPOINT")+os.Getenv("DBTM_GOOGLE_API"), ioData)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var speechBody speechResponse
	if err = json.Unmarshal(body, &speechBody); err != nil {
		return "", err
	}

	// The resp's AudioContent is binary.
	filename := "./temp/output.mp3"
	decodedContent, err := base64.StdEncoding.DecodeString(speechBody.AudioContent)
	if err != nil {
		return "", err
	}
	err = os.WriteFile(filename, decodedContent, 0644)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Audio content written to file: %v\n", filename)

	_, err = exec.Command("mplayer", "./temp/output.mp3").Output()

	if err != nil {
		return "", err
	}

	return string(body), nil
}

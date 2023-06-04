/*
@Author: github.com/imvast
@Date: 06/03/2023
*/

package solver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	captchaService = "CAPSOLVER"
	key            = ""
)

type Solver struct{}

func (s *Solver) SolveCaptcha(session *http.Client) string {
	publicKey := "3daae85e-09ab-4ff6-9f24-e8f4f335e433"
	siteURL := "https://app.revolt.chat"

	if captchaService == "CAPSOLVER" {
		return s.solveGeneric(publicKey, siteURL, session, "https://api.capsolver.com")
	} else if captchaService == "ANTI[CAPTCHA]" {
		return s.solveGeneric(publicKey, siteURL, session, "https://api.anti-captcha.com")
	} else if captchaService == "CAPMONSTER" {
		return s.solveGeneric(publicKey, siteURL, session, "https://api.capmonster.cloud")
	}

	return ""
}

func (s *Solver) solveGeneric(publicKey, siteURL string, session *http.Client, domain string) string {
	taskType := "HCaptchaTaskProxyless"
	if domain != "https://api.capsolver.com" {
		taskType = "HCaptchaTask"
	}
	data1 := map[string]interface{}{
		"clientKey": key,
		"task": map[string]interface{}{
			"type":       taskType,
			"websiteURL": siteURL,
			"websiteKey": publicKey,
			"userAgent":  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36",
			"proxy":      "",
		},
	}
	dataBytes1, _ := json.Marshal(data1)
	resp1, err := session.Post(fmt.Sprintf("%s/createTask", domain), "application/json", bytes.NewReader(dataBytes1))
	if err != nil {
		fmt.Println(err)
		return ""
	}

	var resp1Data map[string]interface{}
	json.NewDecoder(resp1.Body).Decode(&resp1Data)

	if resp1Data["errorId"].(float64) == 0 {
		taskID := resp1Data["taskId"].(string)
		data := map[string]interface{}{
			"clientKey": key,
			"taskId":    taskID,
		}

		dataBytes, _ := json.Marshal(data)
		resp, err := session.Post(fmt.Sprintf("%s/getTaskResult", domain), "application/json", bytes.NewReader(dataBytes))
		if err != nil {
			fmt.Println(err)
			return ""
		}

		var respData map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&respData)

		status := respData["status"].(string)

		for status == "processing" {
			time.Sleep(1 * time.Second)
			resp, err = session.Post(fmt.Sprintf("%s/getTaskResult", domain), "application/json", bytes.NewReader(dataBytes))
			if err != nil {
				fmt.Println(err)
				return ""
			}
			json.NewDecoder(resp.Body).Decode(&respData)
			status = respData["status"].(string)
		}

		if status == "ready" {
			captchaToken := respData["solution"].(map[string]interface{})["gRecaptchaResponse"].(string)
			return captchaToken
		} else {
			return s.SolveCaptcha(session)
		}
	}
	fmt.Println(resp1Data)

	return ""
}

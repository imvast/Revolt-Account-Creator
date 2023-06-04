/*
@Author: github.com/imvast
@Date: 06/03/2023
*/

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"revolt.creator/internal/logging"
	"revolt.creator/internal/mail"
	"revolt.creator/internal/solver"
	"strconv"
	"strings"
	"time"
)

type Revolt struct {
	client  *http.Client
	mailapi *mail.MailGwApi
	headers http.Header
}

func NewRevolt() *Revolt {
	client := &http.Client{Timeout: 30 * time.Second}
	mailapi := mail.NewMailGwApi("", 30)
	headers := http.Header{
		"accept":          []string{"application/json, text/plain, */*"},
		"accept-encoding": []string{"gzip, deflate, br"},
		"accept-language": []string{"en-US,en;q=0.5"},
		"connection":      []string{"keep-alive"},
		"content-type":    []string{"application/json"},
		"host":            []string{"api.revolt.chat"},
		"origin":          []string{"https://app.revolt.chat"},
		"referer":         []string{"https://app.revolt.chat/"},
		"sec-fetch-dest":  []string{"empty"},
		"sec-fetch-mode":  []string{"cors"},
		"sec-fetch-site":  []string{"same-site"},
		"user-agent":      []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/113.0"},
	}
	return &Revolt{
		client:  client,
		mailapi: mailapi,
		headers: headers,
	}
}

func (r *Revolt) register() (string, string) {
	logging.Logger.Debug().
		Strs("domains", r.mailapi.GetDomains()).
		Msg("Available Domains")

	var domain = "hackertales.com"

	var email string
	for {
		email = r.mailapi.GetMail("", "", domain)
		if email == "" {
			continue
		} else {
			break
		}
	}

	logging.Logger.Info().
		Str("email", email).
		Msg("Retrieved Tempmail")

	s := solver.Solver{}
	capkey := s.SolveCaptcha(r.client)
	logging.Logger.Info().
		Str("key", capkey[:50]).
		Msg("Retrieved Captcha")

	passw := "FbghF;2=!Z@u37J"
	payload := map[string]interface{}{
		"email":    email,
		"password": passw,
		"captcha":  capkey,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		return "", ""
	}
	payloadReader := strings.NewReader(string(payloadBytes))

	req, err := http.NewRequest("POST", "https://api.revolt.chat/auth/account/create", payloadReader)
	if err != nil {
		fmt.Println(err)
		return "", ""
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", strconv.Itoa(len(payloadBytes)))

	res, err := r.client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", ""
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return "", ""
	}

	if res.StatusCode == 204 {
		fmt.Printf("(*) Register sent. | %s\n", email)
		return email, passw
	} else if strings.Contains(string(body), "BlockedByShield") {
		logging.Logger.Fatal().
			Str("domain", domain).
			Msg("Domain is blacklisted")
	} else {
		fmt.Printf("%d %s\n", res.StatusCode, string(body))
	}
	return "", ""
}

func (r *Revolt) getEmail() string {
	for {
		time.Sleep(5 * time.Second)
		mails := r.mailapi.FetchInbox()
		for _, emailx := range mails {
			emailxMap := emailx.(map[string]interface{})
			content := r.mailapi.GetMessageContent(emailxMap["id"].(string))
			from := emailxMap["from"].(map[string]interface{})
			fromAddress := from["address"].(string)

			if strings.Contains(fromAddress, "noreply@revolt.chat") {
				contentParts := strings.Split(content, "https://app.revolt.chat/login/verify/")
				if len(contentParts) > 1 {
					id := strings.Split(contentParts[1], "\n")[0]
					return id
				}
			}
		}
	}
}

func (r *Revolt) verifyEmail() string {
	verifyID := r.getEmail()
	fmt.Printf("(~) Verifying... [%s]\n", verifyID)
	url := fmt.Sprintf("https://api.revolt.chat/auth/account/verify/%s", verifyID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	req.Header = r.headers
	req.Header.Set("Content-Length", "0")

	res, err := r.client.Do(req)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		data, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			return ""
		}
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			fmt.Println(err)
			return ""
		}
		accountID := result["ticket"].(map[string]interface{})["account_id"].(string)
		return accountID
	}
	return ""
}

func (r *Revolt) login(mail, passw string) (string, string, string) {
	payload := map[string]interface{}{
		"email":         mail,
		"password":      passw,
		"friendly_name": "firefox on Windows 10",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		return "", "", ""
	}
	payloadReader := strings.NewReader(string(payloadBytes))

	req, err := http.NewRequest("POST", "https://api.revolt.chat/auth/session/login", payloadReader)
	if err != nil {
		fmt.Println(err)
		return "", "", ""
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(payloadBytes)))

	res, err := r.client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", "", ""
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return "", "", ""
	}

	if res.StatusCode == http.StatusOK && strings.Contains(string(body), "Success") {
		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		if err != nil {
			fmt.Println(err)
			return "", "", ""
		}

		accid := result["user_id"].(string)
		sessid := result["_id"].(string)
		sesstokn := result["token"].(string)
		return accid, sessid, sesstokn
	} else {
		fmt.Printf("fail, %s\n", string(body))
		return "", "", ""
	}
}

func (r *Revolt) setupAccount(username, sessionToken string) bool {
	payload := map[string]interface{}{
		"username": username,
	}
	headers := r.headers
	headers.Set("x-session-token", sessionToken)
	headers.Del("content-length")

	payloadBytes, _ := json.Marshal(payload)
	res, err := r.client.Post("https://api.revolt.chat/onboard/complete", "application/json", strings.NewReader(string(payloadBytes)))
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return false
	}

	if strings.Contains(string(data), "UsernameTaken") {
		return r.setupAccount(username+"1", sessionToken)
	}
	if res.StatusCode == 204 {
		return true
	} else {
		fmt.Println(string(data))
		return false
	}
}

func main() {
	logging.Logger.Error().Msg("This version is most likely patched. Join .gg/vast for new version.")

	rand.Seed(time.Now().UnixNano())
	cummer := NewRevolt()
	email, passw := cummer.register()
	cummer.verifyEmail()
	_, _, sessionToken := cummer.login(email, passw)
	username := fmt.Sprintf("imbetterx%s", strings.Join([]string{strconv.Itoa(rand.Intn(10))}, ""))
	cummer.setupAccount(username, sessionToken)
	fmt.Printf("(+) Created: %s:%s:%s\n", username, email, passw)
	f, err := os.OpenFile("./accounts.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf("=== %s ===\nEmail: %s\nPassword: %s\n\n", username, email, passw))
	f2, err := os.OpenFile("./accounts-formatted.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f2.Close()
	f2.WriteString(fmt.Sprintf("%s:%s:%s\n", username, email, passw))

	// abuse stuff — join .gg/vast to get access to this file :D
	// abuser := RevoltAbuse(
	//     cummer.client,
	//     accountID,
	//     sessionID,
	//     sessionToken,
	// )
	// abuser.authorize()
	// abuser.joinGuild()
	// abuser.sendFriend()
	// abuser.massDM()
	// abuser.spamGuild()
}

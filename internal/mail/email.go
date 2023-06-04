/*
@Author: github.com/imvast
@Date: 06/03/2023
*/

package mail

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type MailGwApi struct {
	session *http.Client
	baseURL string
}

func NewMailGwApi(proxy string, timeout int) *MailGwApi {
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return http.ProxyFromEnvironment(req)
			},
		},
	}
	return &MailGwApi{
		session: client,
		baseURL: "https://api.mail.gw",
	}
}

func (m *MailGwApi) GetDomains() []string {
	domains := make([]string, 0)

	response, err := m.session.Get(fmt.Sprintf("%s/domains", m.baseURL))
	if err != nil {
		fmt.Println(err)
		return domains
	}

	if response.StatusCode == 200 {
		var data struct {
			HydraMember []struct {
				Domain string `json:"domain"`
			} `json:"hydra:member"`
		}
		err = json.NewDecoder(response.Body).Decode(&data)
		if err != nil {
			fmt.Println(err)
			return domains
		}

		for _, item := range data.HydraMember {
			domains = append(domains, item.Domain)
		}
	}

	return domains
}

func (m *MailGwApi) GetMail(name, password, domain string) string {
	if name == "" {
		name = generateRandomString(15)
	}
	mail := fmt.Sprintf("%s@%s", name, domain)
	body := map[string]interface{}{
		"address":  mail,
		"password": mail,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		fmt.Println(err)
		return "Email creation error."
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/accounts", m.baseURL), strings.NewReader(string(bodyBytes)))
	if err != nil {
		fmt.Println(err)
		return "Email creation error."
	}

	req.Header.Set("Content-Type", "application/json")

	response, err := m.session.Do(req)
	if err != nil {
		fmt.Println(err)
		return "Email creation error."
	}

	if response.StatusCode == 201 {
		var data struct {
			Token string `json:"token"`
		}
		err = json.NewDecoder(response.Body).Decode(&data)
		if err != nil {
			fmt.Println(err)
			return "Email creation error."
		}

		m.session.Transport.(*http.Transport).Proxy = func(req *http.Request) (*url.URL, error) {
			return http.ProxyFromEnvironment(req)
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", data.Token))

		return mail
	} else {
		bodyBytes, _ := io.ReadAll(response.Body)
		fmt.Println(string(bodyBytes))
		return "Email creation error."
	}
}

func (m *MailGwApi) FetchInbox() []interface{} {
	response, err := m.session.Get(fmt.Sprintf("%s/messages", m.baseURL))
	if err != nil {
		fmt.Println(err)
		return []interface{}{}
	}

	if response.StatusCode == 200 {
		var data struct {
			HydraMember []interface{} `json:"hydra:member"`
		}
		err = json.NewDecoder(response.Body).Decode(&data)
		if err != nil {
			fmt.Println(err)
			return []interface{}{}
		}

		return data.HydraMember
	}

	return []interface{}{}
}

func (m *MailGwApi) GetMessage(messageID string) map[string]interface{} {
	response, err := m.session.Get(fmt.Sprintf("%s/messages/%s", m.baseURL, messageID))
	if err != nil {
		fmt.Println(err)
		return map[string]interface{}{}
	}

	if response.StatusCode == 200 {
		var data map[string]interface{}
		err = json.NewDecoder(response.Body).Decode(&data)
		if err != nil {
			fmt.Println(err)
			return map[string]interface{}{}
		}

		return data
	}

	return map[string]interface{}{}
}

func (m *MailGwApi) GetMessageContent(messageID string) string {
	message := m.GetMessage(messageID)
	return message["text"].(string)
}

func generateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	chars := "abcdefghijklmnopqrstuvwxyz"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

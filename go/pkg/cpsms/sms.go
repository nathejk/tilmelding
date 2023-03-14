package cpsms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type sms struct {
	apikey string
	apiurl string
}

func New(apiurl, apikey string) *sms {
	return &sms{
		apiurl: apiurl,
		apikey: apikey,
	}
}

type cpsmsSingleRecipientRequest struct {
	To      string `json:"to"`
	From    string `json:"from"`
	Message string `json:"message"`
}
type cpsmsSingleRecipientResponse struct {
	Success *struct {
		To   string `json:"to"`
		Cost int    `json:"cost"`
	} `json:"success"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (s *sms) Send(phone, message string) error {
	jsonStr, _ := json.Marshal(cpsmsSingleRecipientRequest{
		To:      "45" + phone,
		From:    "Nathejk",
		Message: message,
	})
	req, err := http.NewRequest("POST", s.apiurl, bytes.NewBuffer(jsonStr))
	req.Header.Set("Authorization", "Basic "+s.apikey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var body cpsmsSingleRecipientResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}
	if body.Error == nil {
		return nil
	}
	return fmt.Errorf("CPSMS Error %q (Code %d)", body.Error.Message, body.Error.Code)
}

package sms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type cpsms struct {
	apikey string
	apiurl string
}

func NewCpsms(host, apikey string) (*cpsms, error) {
	return &cpsms{
		apiurl: "https://" + host + "/v2/send",
		apikey: apikey,
	}, nil
}

func (s *cpsms) Send(phone, message string) error {
	type cpsmsSingleRecipientRequest struct {
		To      string `json:"to"`
		From    string `json:"from"`
		Message string `json:"message"`
	}
	type cpsmsError struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	type cpsmsSingleRecipientResponse struct {
		Success []struct {
			To   string `json:"to"`
			Cost int    `json:"cost"`
		} `json:"success"`
		Error *cpsmsError `json:"error"`
	}

	jsonStr, _ := json.Marshal(cpsmsSingleRecipientRequest{
		To:      "45" + phone,
		From:    "Nathejk",
		Message: message,
	})
	req, err := http.NewRequest("POST", s.apiurl, bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}
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
	return fmt.Errorf("CPSMS Error %d: %q", body.Error.Code, body.Error.Message)
}

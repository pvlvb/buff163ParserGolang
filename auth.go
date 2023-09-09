package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

type LoginResponse struct {
	Token string `json:"token"`
}

// Authenticate gets the JWT token from the backend.
func Authenticate() (string, error) {
	resp, err := http.Post("http://localhost/auth/signin", "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("failed to authenticate")
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	var loginResponse LoginResponse
	json.Unmarshal(bodyBytes, &loginResponse)

	return loginResponse.Token, nil
}

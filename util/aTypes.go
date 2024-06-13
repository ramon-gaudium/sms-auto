package util

import (
	"encoding/json"
	"strings"
)

const (
	CD_INVALID_JSON        int    = 1
	MSG_INVALID_JSON_READ  string = "Informação inválida para leitura"
	MSG_INAVLID_JSON_WRITE string = "Informação inválida para escrita"

	/*MSG_SMS_SENT string = "SMS enviado"
	MSG_SMS_WAIT string = "Aguarde a chegada do SMS"
	MSG_SMS_EXPIRED string = "Envio expirado, tente novamente"

	CD_SMS_FAILED int = 100
	MSG_SMS_FAILED string = "Envio do SMS falhou"*/
)

type TSendRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	AppId       string `json:"appId"`
	Bandeira    string `json:"bandeira"`
	Content     string `json:"content"`
}

func NewSendRequest(content []byte) (result TSendRequest, err error) {
	err = json.Unmarshal(content, &result)
	if err == nil {
		result.AppId = strings.TrimSpace(result.AppId)
	}
	return result, err
}

//TVerifyRequest ------
type TVerifyRequest struct {
	PhoneNumber    string `json:"phoneNumber"`
	Bandeira       string `json:"bandeira"`
	AppId          string `json:"appId"`
	ValidationCode string `json:"validationCode"`
}

func NewVerifyRequest(content []byte) (result TVerifyRequest, err error) {
	err = json.Unmarshal(content, &result)
	return result, err
}

//TFindTokenRequest ------
type TFindTokenRequest struct {
	Token string `json:"token"`
}

func NewFindTokenRequestFromJson(content []byte) (result TFindTokenRequest, err error) {
	err = json.Unmarshal(content, &result)
	return result, err
}

type TFindTokenResponse struct {
	PhoneNumber    string `json:"phoneNumber"`
	ValidationCode string `json:"validationCode"`
}

/*func NewFindTokenResponseFromJson(content []byte) (result TFindTokenResponse, err error) {
	err = json.Unmarshal(content, &result)
	return result, err
}*/

func NewFindTokenResponseToJson(phoneNumber string, validationCode string) (string, error) {
	bts, err := json.Marshal(TFindTokenResponse{PhoneNumber: phoneNumber, ValidationCode: validationCode})
	return string(bts), err
}

//------

type TResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Data    string `json:"data"`
}

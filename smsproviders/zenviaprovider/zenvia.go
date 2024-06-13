package zenviaprovider

import (
	"encoding/json"
	"fmt"
	"gaudium.com.br/gaudiumsoftware/sms/smsproviders"
	"github.com/valyala/fasthttp"
	"log"
	"strconv"
)

const (
	ZenviaProviderName 					= "Zenvia"
	zenviaAppKey 						= "hKp94crjv9OF3UGrCpSXUJw1-UYHhRvLKNLt"
	zenviaSendURI 						= "https://api.zenvia.com/v1/channels/sms/messages"
	//zenviaVerifyURI = "%s"
	zenviaSendSmsTemplate              	=  `{"from":"Gaudium","to":"%s","contents":[{"type": "text","text": "%s"},{"type": "text","text": "%s"}]}`
	zenviaSendVerificationJsonTemplate 	= `{"from":"Gaudium","to":"%s","contents":[{"type": "text","text": "%s"},{"type": "text","text": "%s"}]}`
	zenviaVerifyResponseJsonTemplate   	= `{"method": "sms","sms": { "code": "%s" }}`

	ZENVIA_VERIFY_SUCCESS      = "SUCCESSFUL"
	ZENVIA_FAILED              = 1
	ZENVIA_PARSE_ERROR_CODE    = 2
	ZENVIA_SEND_VR_ERROR_CODE  = 10 //Envio do pedido de verificação
	ZENVIA_SEND_SMS_ERROR_CODE = 11 //Envio de SMS
	ZENVIA_VERIFY_ERROR_CODE   = 20 //
)

type TZenviaSendMessageResponse struct {
	Message string
	Status string
}

type ZenviaSmsVerifier struct {
	providerName string
	appKey string
	sendUri string
}

func NewZenviaSmsVerifier() smsproviders.SmsProviderIntf {
	result := &ZenviaSmsVerifier{ZenviaProviderName,zenviaAppKey, zenviaSendURI}
	return result
}

func (s *ZenviaSmsVerifier) ProviderName() string {
	return s.providerName
}

func (s *ZenviaSmsVerifier) SendVerificationRequest(phoneNumber string, content string, hashCode string) (result smsproviders.SmsResult) {
	log.Println("SmsSendMessage: " + content + " / " + hashCode)
	return s.SendMessageRequest(phoneNumber, content, hashCode)
}

func (s *ZenviaSmsVerifier) CheckSendVerificationResponse(content []byte) (result smsproviders.SmsResult) {
	result = s.CheckSendVerificationResponse(content)
	log.Println("CheckSendVerificationResponse: " + strconv.FormatBool(result.IsSuccess) + ":" + result.Msg + ":" + fmt.Sprintf("%v", result.Data))
	return result
}

func (s *ZenviaSmsVerifier) SendMessageRequest(phoneNumber string, content string, hashCode string) (result smsproviders.SmsResult) {
	log.Print("SmsSendVerificationRequest")

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(s.sendUri)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.Header.Add("X-API-TOKEN", zenviaAppKey)
	req.SetBodyString(fmt.Sprint(zenviaSendVerificationJsonTemplate, phoneNumber, content, hashCode))
	resp := fasthttp.AcquireResponse()
	client := &fasthttp.Client{}
	if err := client.Do(req, resp); err != nil {
		result = *smsproviders.NewSmsResult(smsproviders.NoSuccess, ZENVIA_SEND_SMS_ERROR_CODE, err.Error(), "")
	} else {
		bodyBytes := resp.Body()
		result = s.CheckSendMessageResponse(bodyBytes)
	}
	return result
}

func (s *ZenviaSmsVerifier) CheckSendMessageResponse(content []byte) (result smsproviders.SmsResult) {
	var vResp TZenviaSendMessageResponse
	err := json.Unmarshal(content, &vResp)
	if err == nil {
		if vResp.Status == "***" {
			result = *smsproviders.NewSmsResult(smsproviders.Success, smsproviders.SuccessCode, vResp.Message, vResp.Status)
		} else {
			result = *smsproviders.NewSmsResult(smsproviders.NoSuccess, ZENVIA_SEND_SMS_ERROR_CODE, vResp.Message, vResp.Status)
		}
	}
	return result
}

func (s *ZenviaSmsVerifier) VerifyRequest(phoneNumber string, sentCode string, receivedCode string) (result smsproviders.SmsResult) {
	log.Print("SmsVerifyRequest")

	if sentCode != receivedCode {
		return *smsproviders.NewSmsResult(smsproviders.Success, ZENVIA_FAILED, "Código inválido", "")
	} else {
		return *smsproviders.NewSmsResult(smsproviders.Success, smsproviders.SuccessCode, "Validado com sucesso", "")
	}
}

func (s *ZenviaSmsVerifier) CheckVerifyResponse(content []byte) (result smsproviders.SmsResult) {
	return s.CheckSendMessageResponse(content)
}

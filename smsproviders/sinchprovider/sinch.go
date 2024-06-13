package sinchprovider

import (
	"encoding/json"
	"fmt"
	"gaudium.com.br/gaudiumsoftware/sms/smsproviders"
	"gaudium.com.br/gaudiumsoftware/sms/util"
	"github.com/valyala/fasthttp"
	"strconv"
	"strings"
	"time"
)

const (
	SinchProviderName				  = "Sinch"
	// sinchAppKey                       = "9da910fc-e2ea-4a4c-90cc-10cb30d330e4"
	// sinchAppKey                       = "YjVjNThlZjUtOTBkNC00MzkwLWE3NDEtZTk1NWU5YmNkNDZjOm5qSW5oM0R3TUVxRzNqNm9qbE5Kc3c9PQ=="
	sinchAppKey                       = "YjVjNThlZjUtOTBkNC00MzkwLWE3NDEtZTk1NWU5YmNkNDZjOm5qSW5oM0R3TUVxRzNqNm9qbE5Kc3c9PQ=="
	sinchSendURI                      = "https://verificationapi-v1.sinch.com/verification/v1/verifications"
	sinchVerifyURI                    = "https://verificationapi-v1.sinch.com/verification/v1/verifications/number/%s"
	sinchSendSmsTemplate			  = `{"from": "Gaudium","to": ["%s"],"body": "%s \n(%s)"}'"`
	sinchSendVerificationJsonTemplate = `{"identity":{"type":"number","endpoint":"%s"},"method":"sms","smsOptions":{"applicationHash":"%s"}}`
	sinchVerifyJsonTemplate 		  = `{"method": "sms","sms": { "code": "%s" }}`

	SinchVerifySuccess    = "SUCCESSFUL"
	SinchParseErrorCode   = 1
	SinchSendVrErrorCode  = 10 //Erro no envio do pedido de verificação
	SinchSendSmsErrorCode = 11 //Erro no envio de SMS
	SinchVerifyErrorCode  = 20 //Erro na verificação do código
)


//TSinchSendRequest ------
type TIdentity struct {
	Type string `json:"type"`
	Endpoint string `json:"endpoint"`
}

type TSmsOptions struct {
	ApplicationHash string `json:"applicationHash"`
}

//***
type TSinchSendRequest struct {
	Identity TIdentity `json:"identity"`
	Method string `json:"method"`
	SmsOptions TSmsOptions `json:"smsOptions"`
}
//------

type _SmsStruct struct {
	Template string `json:"template"`
	InterceptionTimeout int `json:"interceptionTimeout"`
}

//***
type TSinchSendResponse struct {
	Id string `json:"id"`
	Sms _SmsStruct `json:"sms"`
	Method string `json:"method"`
	Status string `json:"status"`
	ErrorCode int64 `json:"errorCode"`
	Message string `json:"message"`
	Reference string `json:"reference"`
}

func NewSinchSendResponse(content []byte) (result TSinchSendResponse, err error) {
	err = json.Unmarshal(content, &result)
	return result, err
}

func translateMessage(errorCode int64) string {
	switch errorCode {
	//BadRequest
	case 40001: return "Número inválido" 								//ParameterValidation
	case 40002: return "Parâmetro inválido" 							//MissingParameter
	case 40003: return "Código inválido" 								//InvalidRequest
	case 40004: return "Não autorizado" 								//InvalidAuthorizationKey
	case 40005: return "Formato não reconhecido. Não possui `+`" 		//NumberMissingLeadingPlus
	//Unauthorized
	case 40100, 40101, 40102, 40103,									//AuthorizationHeader, TimestampHeader, InvalidSignature, AlreadyAuthorized
		40104, 40105, 40106, 40107,										//AuthorizationRequired, Expired, UserBarred, InvalidAuthorization
		40108: return "Näo autorizado"									//InvalidCredentials
	//PaymentRequired
	case 40200: return "Não foi possível verificar o número de telefone no momento. Por favor, tente mais tarde."  //NotEnoughCredit
	//Forbidden
	case 40300, 40301, 40302, 40303: return "Acesso inválido"			//ForbiddenRequest, InvalidScheme, InsufficientPrivileges, RestrictedAction,
	//NotFound
	case 40400: return "Recurso não encontrado"							//ResourceNotFound
	//Conflict
	case 40900: return "Conflito de pedido"								//RequestConflict
	//UnprocessableEntity
	case 42200: return "Erro de configuração" 							//ApplicationConfiguration
	case 42201: return "Não disponível"									//Unavailable
	case 42202: return "Resposta da chamada de retorno inválida"		//InvalidCallbackResponse
	//TooManyRequests
	case 42900: return "Capacidade excedida"							//CapacityExceeded
	case 42901: return "Limite de velocidade atingido"					//VelocityConstraint
	//InternalServerError
	case 50000: return "Erro interno"									//InternalError
	//NotImplemented
	case 50100: return "Método não implementado"						//MethodNotImplemented
	case 50101: return "Status não implementado"						//StatusNotImplemented
	//ServiceUnavailable
	case 50300: return "Serviço temporariamente indisponível"			//TemporaryDown
	case 50301: return "Erro de configuração"							//ConfigurationError
	default:
		return "Erro " + strconv.FormatInt(int64(errorCode), 10) + ". Entre em contato com um dos nossos antendentes."
	}
}

type _SmsStruct2 struct {
	Code string `json:"code"`
}

type TSinchVerifyRequest struct {
	Method string `json:"method"`
	Sms _SmsStruct2 `json:"sms"`
}

type TSinchVerifyResponse struct {
	//Success fields
	Id string `json:"id"`
	Method string `json:"method"`
	Status string `json:"status"`
	//Error fields
	ErrorCode int64 `json:"errorCode"`
	Message string `json:"message"`
}

//TSinchVerifyResponse ------
func NewSinchVerifyResponse(content []byte) (result TSinchVerifyResponse, err error) {
	err = json.Unmarshal(content, &result)
	return result, err
}
//------

//TSendResponse ------
type _TSinchSendResponseSms struct {
	Template string `json:"template"`
	InterceptionTimeout int `json:"interceptionTimeout"`
}

type TSendResponse struct {
	Id     string                 `json:"id"`
	Sms    _TSinchSendResponseSms `json:"sms"`
	Method string                 `json:"method"`
}

//------

type SinchSmsVerifier struct {
	providerName string
	appKey string
	sendUri string
	verifyUri string
}

func NewSinchSmsVerifier() smsproviders.SmsProviderIntf {
	result := &SinchSmsVerifier{SinchProviderName, sinchAppKey, sinchSendURI, sinchVerifyURI}
	return result
}

func (s *SinchSmsVerifier) ProviderName() string {
	return s.providerName
}

func (s *SinchSmsVerifier) SendVerificationRequest(phoneNumber string, content string, hashCode string) (result smsproviders.SmsResult) {
	util.LogD("SendVerificationRequest.1: " + phoneNumber + ":" + content + ":" + hashCode)

	if !strings.HasPrefix(phoneNumber, "+55") {
		util.LogD("Validação de SMS sem os parâmetros corretos - Telefone: " + phoneNumber + " - Hashcode: " + hashCode)
		resultaError := *smsproviders.NewSmsResult(smsproviders.Success, smsproviders.SuccessCode, "Pedido de verificação enviado com sucesso", "")
		return resultaError
	}

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(s.sendUri)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	t := time.Now()
	req.Header.Add("Date", t.Format("2019-12-20T19:49:00.0000000Z"))
	util.LogD(string(req.Header.Peek("Date")))
	// req.Header.Add("Authorization", "Application " + s.appKey)
	req.Header.Add("Authorization", "Basic " + s.appKey)
	req.Header.Add("Accept-Language", "pt-BR")
	if content != "" {
		content = ""
	}
	reqBody := fmt.Sprintf(sinchSendVerificationJsonTemplate, phoneNumber, hashCode);
	util.LogD("SendVerificationRequest.2: " + reqBody)
	req.SetBodyString(reqBody)
	resp := fasthttp.AcquireResponse()
	client := &fasthttp.Client{}
	if err := client.Do(req, resp); err != nil {
		util.LogE("SendVerificationRequest.3 (falha): " + err.Error())
		result = *smsproviders.NewSmsResult(smsproviders.NoSuccess, SinchSendVrErrorCode, err.Error(), "")
	} else {
		util.LogD("SendVerificationRequest.4 (sucesso)")
		result = s.CheckSendVerificationResponse(resp.Body())
	}
	util.LogD("SendVerificationRequest.5: " + strconv.FormatBool(result.IsSuccess) + ":" + result.Msg + ":" + fmt.Sprintf("%v", result.Data))
	return result
}

func (s *SinchSmsVerifier) CheckSendVerificationResponse(content []byte) (result smsproviders.SmsResult) {
	util.LogD("CheckSendVerificationResponse.1: " + string(content))

	var vSendResp TSinchSendResponse
	err := json.Unmarshal(content, &vSendResp)
	util.LogD("CheckSendVerificationResponse.unmarshal.id: " + vSendResp.Id)
	if err == nil {
		util.LogD("CheckSendVerificationResponse.vSendResp.Status: " + vSendResp.Status)
		if vSendResp.Id != "" {
			util.LogD("CheckSendVerificationResponse.2: sucesso")
			result = *smsproviders.NewSmsResult(smsproviders.Success, smsproviders.SuccessCode, "Pedido de verificação enviado com sucesso", vSendResp.Id)
		} else {
			util.LogD("CheckSendVerificationResponse.3: falha")
			vSendResp.Message = translateMessage(vSendResp.ErrorCode)
			result = *smsproviders.NewSmsResult(smsproviders.NoSuccess, SinchSendVrErrorCode, vSendResp.Message, strconv.FormatInt(vSendResp.ErrorCode, 10) + " - " + vSendResp.Reference)
		}
	}
	util.LogD("CheckSendVerificationResponse.4: " + strconv.FormatBool(result.IsSuccess) + ":" + result.Msg + ":" + fmt.Sprintf("%v", result.Data))
	return result
}

	/*func (s *SinchSmsVerifier) SendMessageRequest(phoneNumber string, content string, hashCode string) (result smsproviders.SmsResult) {
	util.LogD("SendMessageRequest: " + phoneNumber + ":" + content + ":" + hashCode)

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(s.sendUri)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	t := time.Now()
	req.Header.Add("Date", t.Format("2019-12-20T19:49:00.0000000Z"))
	req.Header.Add("Authorization", "Application " + s.appKey)
	req.Header.Add("Accept-Language", "pt-BR")
	req.SetBodyString(fmt.Sprintf(sinchSendSmsTemplate, phoneNumber, content, hashCode))
	resp := fasthttp.AcquireResponse()
	client := &fasthttp.Client{}
	if err := client.Do(req, resp); err != nil {
		result = *smsproviders.NewSmsResult(smsproviders.NoSuccess, err.Error(), "")
	} else {
		bodyBytes := resp.Body()
		result = s.CheckSendMessageResponse(bodyBytes)
	}
	util.LogD("SendMessageRequest: " + strconv.FormatBool(result.IsSuccess) + ":" + result.Msg + ":" + fmt.Sprintf("%v", result.Data))
	return result
}*/

/*func (s *SinchSmsVerifier) CheckSendMessageResponse(content []byte) (result smsproviders.SmsResult) {
	var vResp TSinchVerifyResponse;
	err := json.Unmarshal(content, &vResp)
	if err == nil {
		if vResp.Status == SINCH_SMS_SUCCESS {
			result = *smsproviders.NewSmsResult(SINCH_SUCCESS, "SMS enviado com sucesso", vResp.Id)
		} else {
			result = *smsproviders.NewSmsResult(SINCH_SEND_SMS_ERROR_CODE, vResp.Message + "(" + vResp.ErrorCode + ")", vResp.Id)
		}
	}
	util.LogD("SendVerificationResponse: " + string(result.IsSuccess) + ":" + result.Msg + ":" + result.Data)
	return result
}*/

func (s *SinchSmsVerifier) VerifyRequest(phoneNumber string, sentCode string, receivedCode string) (result smsproviders.SmsResult) {
	util.LogD("SmsVerifyRequest")

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(fmt.Sprintf(s.verifyUri, phoneNumber))
	req.Header.SetMethod("PUT")
	req.Header.SetContentType("application/json")
	t := time.Now()
	req.Header.Add("Date", t.Format("2006-02-01T15:04:05.0000000Z"))
	util.LogD(string(req.Header.Peek("Date")))
	// req.Header.Add("Authorization", "Application " + s.appKey)
	req.Header.Add("Authorization", "Basic " + s.appKey)
	req.Header.Add("Accept-Language", "pt-BR")
	req.SetBodyString(fmt.Sprintf(sinchVerifyJsonTemplate, receivedCode))
	resp := fasthttp.AcquireResponse()
	client := &fasthttp.Client{}
	if err := client.Do(req, resp); err != nil {
		result = *smsproviders.NewSmsResult(smsproviders.NoSuccess, SinchVerifyErrorCode, err.Error(), "")
	} else {
		result = s.CheckVerifyResponse(resp.Body())
	}
	return result
}

func (s *SinchSmsVerifier) CheckVerifyResponse(content []byte) (result smsproviders.SmsResult) {
	var vResp TSinchVerifyResponse
	err := json.Unmarshal(content, &vResp)
	if err == nil {
		if vResp.Status == SinchVerifySuccess {
			result = *smsproviders.NewSmsResult(smsproviders.Success, smsproviders.SuccessCode, "Validado com sucesso", "")
		} else {
			vResp.Message = translateMessage(vResp.ErrorCode)
			if vResp.ErrorCode == 40003 {
				result = *smsproviders.NewSmsResult(smsproviders.NoSuccess, SinchVerifyErrorCode, vResp.Message, vResp.Id)
			} else {
				result = *smsproviders.NewSmsResult(smsproviders.NoSuccess, SinchVerifyErrorCode, vResp.Message + " ("+strconv.FormatInt(vResp.ErrorCode, 10) + ")", vResp.Id)
			}
		}
	} else {
		result = *smsproviders.NewSmsResult(smsproviders.NoSuccess, SinchParseErrorCode, err.Error(), "")
	}
	return result
}

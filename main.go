package main

import (
	"encoding/json"
	"fmt"
	db "gaudium.com.br/gaudiumsoftware/sms/redisDb"
	"gaudium.com.br/gaudiumsoftware/sms/smsproviders"
	"gaudium.com.br/gaudiumsoftware/sms/smsproviders/sinchprovider"
	"gaudium.com.br/gaudiumsoftware/sms/smsproviders/zenviaprovider"
	"gaudium.com.br/gaudiumsoftware/sms/util"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"log"
	"os"
)

const (
	defaultPort          = 8081
	rootEndpoint         = "/api/sms"
	rootInternalEndpoint = "/api/sms-internal"

	//Verification SMS
	verificationEndPoint = rootEndpoint + "/verification"
	requestEndpoint      = verificationEndPoint + "/send"
	verifyEndpoint       = verificationEndPoint + "/verify"

	//Standard SMS
	messagingEndPoint      = rootEndpoint + "/messaging"
	RequestSendSmsEndpoint = messagingEndPoint + "/sendSms"

	//Backend service (internal only)
	findTokenEndpoint      = rootInternalEndpoint + "/findToken"
	changeProviderEndpoint = rootInternalEndpoint + "/provider"
)

var defaultProvider = sinchprovider.SinchProviderName

/*
- Enviar SMS de verificação
  - Backend
  	- Sinch
		- API Verificação
			- Enviar - (ok)
    		- Verificar - (ok)
				- Envio da confirmação ao backend PHP (ok)
		- API SMS
			- Enviar (testar)
  - Android
    - Tela de validação
		- Solicitação de verificação ao backend (ok)
		- Aviso de espera e timeout (ok)
		- Recebimento do SMS e extração do código (ok)
		- Envio do código recebido ao backend (ok)
*/

func NewSmsProvider(ProviderName string) smsproviders.SmsProviderIntf {
	switch ProviderName {
	case sinchprovider.SinchProviderName:
		return sinchprovider.NewSinchSmsVerifier()
	case zenviaprovider.ZenviaProviderName:
		return zenviaprovider.NewZenviaSmsVerifier()
	default:
		return nil
	}
}

func newOkResponse(result smsproviders.SmsResult) string {
	return newOkResponseFromValues(result.Msg, fmt.Sprintf("%v", result.Data))
}

func newOkResponseFromValues(Msg string, Data string) string {
	response := util.TResponse{smsproviders.Success, smsproviders.SuccessCode, Msg, Data}
	bts, err := json.Marshal(response)
	if err == nil {
		return string(bts)
	} else {
		return newErrorResponseFromValues(util.CD_INVALID_JSON, util.MSG_INVALID_JSON_READ, err.Error())
	}
}

func newErrorResponse(result smsproviders.SmsResult) string {
	return newErrorResponseFromValues(result.Code, result.Msg, fmt.Sprintf("%v", result.Data))
}

func newErrorResponseFromValues(Code int, Err string, Data string) (result string) {
	response := util.TResponse{Success: false, Code: Code, Msg: Err, Data: Data}
	bts, err := json.Marshal(response)
	if err == nil {
		return string(bts)
	} else {
		bts, _ = json.Marshal(util.TResponse{Success: smsproviders.NoSuccess, Msg: err.Error(), Data: ""})
		return string(bts)
	}
}

func requestVerificationHandler(ctx *fasthttp.RequestCtx) {
	sendReq, err := util.NewSendRequest(ctx.Request.Body())
	if err == nil {
		//Descomentar para teste
		//util.SendResponse(ctx, fasthttp.StatusOK, newOkResponseFromValues("OK", ""))
		//return
		util.LogD("requestVerificationHandler: pn: " + sendReq.PhoneNumber)
		_ = db.MovePossibleFailedRequest(&sendReq.PhoneNumber, &sendReq.Bandeira)
		reqData := db.NewRequestData("", "", sendReq.PhoneNumber, sendReq.Bandeira, "", "", "")
		//Grava as primeiras informações do pedido de envio
		reqData, err = db.WriteRequest(reqData)
		if err != nil {
			util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(db.RedisWriteError, err.Error(), ""))
			return
		}
		provider := NewSmsProvider(defaultProvider)
		result := provider.SendVerificationRequest(sendReq.PhoneNumber, sendReq.Content, sendReq.AppId)
		if result.IsSuccess == smsproviders.Success {
			sq := db.AccountSMS(sendReq.Bandeira)
			if sq > "" {
				reqData.Sq = sq
			} else {
				util.SendResponse(ctx, fasthttp.StatusOK, newOkResponseFromValues(result.Msg, ""))
				return
			}
			util.LogD("requestVerificationHandler (Success): " + result.Msg + " / " + fmt.Sprintf("%v", result.Data))
			reqData.SmsId = fmt.Sprintf("%v", result.Data)
			//Grava as informações restantes após o peido de envio ter tido sucesso
			reqData, err = db.WriteRequest(reqData)
			if err == nil {
				util.SendResponse(ctx, fasthttp.StatusOK, newOkResponseFromValues(result.Msg, ""))
			} else {
				util.LogD("requestVerificationHandler.WriteRequest (NoSuccess): " + err.Error())
				util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(db.RedisWriteError, err.Error(), ""))
			}
		} else {
			util.LogD("requestVerificationHandler (NoSuccess): " + result.Msg)
			util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponse(result))
		}
	} else {
		util.LogD("requestVerificationHandler (Error): " + err.Error())
		util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(util.CD_INVALID_JSON, util.MSG_INVALID_JSON_READ, err.Error()))
	}
}

/*func requestSmsHandler(ctx *fasthttp.RequestCtx) {
	var smsReq util.TSendRequest
	err := json.Unmarshal(ctx.Request.Body(), &smsReq)
	if err == nil {
		provider := NewSmsProvider(defaultProvider)
		util.LogD(smsReq.PhoneNumber)
		util.LogD(smsReq.Content)
		result := provider.SendMessageRequest(smsReq.PhoneNumber, smsReq.Content, smsReq.AppId)
		if result.IsSuccess == smsproviders.Success {
			util.SendResponse(ctx, fasthttp.StatusOK, newOkResponse(result))
		} else {
			util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponse(result))
		}
	} else {
		util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(util.CD_INVALID_JSON, util.MSG_INVALID_JSON_READ, err.Error()))
	}
}*/

func verifyHandler(ctx *fasthttp.RequestCtx) {
	vReq, err := util.NewVerifyRequest(ctx.Request.Body())
	if err == nil {
		util.LogD("VerifyRequest: " + vReq.PhoneNumber + " / " + vReq.ValidationCode)
		provider := NewSmsProvider(defaultProvider)
		util.LogD(vReq.PhoneNumber)
		util.LogD(vReq.ValidationCode)
		result := provider.VerifyRequest(vReq.PhoneNumber, vReq.ValidationCode, vReq.ValidationCode)
		if result.IsSuccess == smsproviders.Success {
			util.LogD("VerifyResponse (Success): " + result.Msg)
			var reqData *db.RequestData
			reqData, err = db.ReadRequest(&vReq.PhoneNumber, &vReq.Bandeira)
			if reqData == nil {
				util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(db.RedisNotFoundError, "Pedido inválido ou expirou", ""))
			} else {
				respData := db.NewResponseData(reqData.Key, reqData.IdPedidoEnvio, vReq.PhoneNumber, vReq.Bandeira, reqData.Sq, reqData.SmsId, vReq.ValidationCode, reqData.TimestampSend, "")
				var dataResult *db.ResponseData
				dataResult, err = db.WriteResponse(&respData)
				if (err == nil) && (dataResult.Key != "") {
					db.DiscardRequestFields(&vReq.PhoneNumber, &vReq.Bandeira) //si e sq passaram a estar em rs, então descarta de rq
					util.SendResponse(ctx, fasthttp.StatusOK, newOkResponseFromValues("Validado com sucesso", dataResult.SmsId))
				} else {
					util.LogD("VerifyResponse (NoSuccess): token não encontrado")
					util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(db.RedisWriteError, "Não foi possível gerar o token", err.Error()))
				}
			}
		} else {
			util.LogD("VerifyResponse (NoSuccess): " + result.Msg)
			util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponse(result))
		}
	} else {
		util.LogD("VerifyResponse (Error): " + err.Error())
		util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(util.CD_INVALID_JSON, util.MSG_INVALID_JSON_READ, err.Error()))
	}
}

func changeProviderHandler(ctx *fasthttp.RequestCtx) {
	providerParam := string(ctx.QueryArgs().Peek("provider"))
	authKeyParam := string(ctx.QueryArgs().Peek("key"))
	if providerParam != "" {
		if providerParam == "?" {
			util.SendResponse(ctx, fasthttp.StatusOK, newOkResponseFromValues("Provider: " +defaultProvider, ""))
		} else if authKeyParam == util.AppCfg.ChangeProviderKey {
			newProv := NewSmsProvider(providerParam)
			if newProv != nil {
				defaultProvider = newProv.ProviderName()
				util.SendResponse(ctx, fasthttp.StatusOK, newOkResponseFromValues("Novo provider ativado: " +defaultProvider, ""))
			} else {
				util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(-1, "Provider inválido", ""))
			}
		} else {
			util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(-1, "Não autorizado", ""))
		}
	} else {
		util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(util.CD_INVALID_JSON, util.MSG_INVALID_JSON_READ, ""))
	}
}

func findTempTokenHandler(ctx *fasthttp.RequestCtx) {
	fReq, err := util.NewFindTokenRequestFromJson(ctx.Request.Body())
	if err == nil {
		util.LogD("FindToken: " + fReq.Token)
		phoneNumber, validationCode, err := db.FindTempToken(fReq.Token)
		if err == nil {
			if (phoneNumber == "") || (validationCode == "") {
				util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(db.RedisNotFoundError, "Token inválido ou expirado", ""))
			} else {
				tokenDataStr, tokenErr := util.NewFindTokenResponseToJson(phoneNumber, validationCode)
				if (tokenErr == nil) && (tokenDataStr != "") {
					util.SendResponse(ctx, fasthttp.StatusOK, newOkResponseFromValues("OK", tokenDataStr))
				} else {
					var dataStr string
					if tokenErr != nil {
						dataStr = tokenErr.Error()
					} else {
						dataStr = ""
					}
					util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(util.CD_INVALID_JSON, util.MSG_INAVLID_JSON_WRITE, dataStr))
				}
			}
		} else {
			util.LogD("findToken (NoSuccess): " + err.Error())
			util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(db.RedisNotFoundError, "A leitura do token falhou.", err.Error()))
		}
	} else {
		util.LogD("FindToken (Error): " + err.Error())
		util.SendResponse(ctx, fasthttp.StatusOK, newErrorResponseFromValues(util.CD_INVALID_JSON, util.MSG_INVALID_JSON_READ, err.Error()))
	}
}

func main() {
	var f *os.File

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in main(): ", r)
		}
		if f != nil {
			f.Close()
		}
	}()

	f = util.InitLog(util.AppCfg.LogFileName)
	util.LogD("Using config: " + util.DefaultConfigPath + util.DefaultConfigFile)
	util.LogD("Using log: " + util.AppCfg.LogFileName)

	connFailed, errRedis := db.SetupRedisPool()
	if errRedis != nil {
		log.Fatal(errRedis.Error())
	}

	if connFailed {
		log.Fatal("Conexão com o Redis falhou.")
	}

	util.LogD("---endpoints---")
	fastHTTPRouter := router.New()
	fastHTTPRouter.POST(requestEndpoint, requestVerificationHandler)
	util.LogD(requestEndpoint)
	/*fastHTTPRouter.POST(requestSendSmsEndpoint, requestSmsHandler)
	util.LogD(requestSendSmsEndpoint)*/
	fastHTTPRouter.POST(verifyEndpoint, verifyHandler)
	util.LogD(verifyEndpoint)
	fastHTTPRouter.POST(findTokenEndpoint, findTempTokenHandler)
	util.LogD(findTokenEndpoint)
	fastHTTPRouter.GET(changeProviderEndpoint, changeProviderHandler)
	util.LogD(changeProviderEndpoint)
	util.LogD("---endpoints---")
	serverAddr := fmt.Sprint(":", util.AppCfg.NetworkOptions.ListeningPort)
	util.LogD(serverAddr)
	requestHandler := fasthttp.CompressHandlerLevel(fastHTTPRouter.Handler, fasthttp.CompressBestCompression)
	errorHandlingRootRequest := fasthttp.ListenAndServe(serverAddr, requestHandler)
	if errorHandlingRootRequest != nil {
		util.LogD(errorHandlingRootRequest.Error())
	}
	util.LogD("End")
	log.Fatal(errorHandlingRootRequest.Error())
}

func init() {
	util.AppCfg = util.NewConfig(db.DefRedisConnectionString, db.DefRedisPoolSize, db.DefRedisDialTimeout,
		util.DefaultHttpPort,
		util.DefaultMaxSmsRequestsPerPhone, util.DefaultResendWaitSecondsAfterTriesLimitReached)
	util.LoadConfig(util.DefaultConfigPath + util.DefaultConfigFile, &util.AppCfg)

	//util.DefaultMaxSmsRequestsPerPhone = util.AppCfg.SmsOptions.MaxSmsRequestsPerPhone
	//util.DefaultResendWaitSecondsAfterTriesLimitReached = util.AppCfg.SmsOptions.SmsSecureRequestIntervalInMinutes

	util.SetLogEnabled(util.AppCfg.LogEnabled)
	util.SetLogOptions(util.AppCfg.LogOptions)
	util.PrintConfig(util.AppCfg, false)
	util.PrintConfig(util.AppCfg, true)
}

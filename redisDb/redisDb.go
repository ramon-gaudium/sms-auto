package redisDb

import (
	"encoding/json"
	"errors"
	"fmt"
	"gaudium.com.br/gaudiumsoftware/sms/util"
	"github.com/mediocregopher/radix/v3"
	"github.com/valyala/fasthttp"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
)

const (
	RedisWriteError    int = 20001
	RedisNotFoundError int = 20404

	DefRedisConnectionString     = "kdev-redis.gaudium.lan:6379"
	DefRedisPoolSize         int = 20
	DefRedisDialTimeout      int = 30

	logServiceSaveMethod = "/save"
)

var redisClient *radix.Pool
var errRedis error
var redisByPass bool
var requestTTL int64 = 6 * 30 * 24 * 60 * 60 //6 meses - Tempo que o request fica armazenado no Redis
var tempTokenTTL int64 = 30 * 60             //3 minutos - Depois de validado o SMS, o token fica disponível %d minutos no Redis para ser consumido pelo cadastro no PHP

type RequestData struct {
	Key           string
	IdPedidoEnvio string
	PhoneNumber   string
	Bandeira      string
	Sq            string
	SmsId         string
	TimestampSend string
}

type ResponseData struct {
	Key              string
	IdPedidoEnvio    string
	PhoneNumber      string
	Bandeira         string
	Sq               string
	SmsId            string
	ValidationCode   string
	TimestampSend    string
	TimestampReceive string
}

func NewRequestData(key string, idPedidoEnvio string, phoneNumber string, bandeira string, sq string, smsId string, tsSend string) RequestData {
	return RequestData{key, idPedidoEnvio, phoneNumber, bandeira, sq, smsId, tsSend}
}

func NewResponseData(key string, idPedidoEnvio string, phoneNumber string, bandeira string, sq string, smsId string, validationCode string, tsSend string, tsReceive string) ResponseData {
	return ResponseData{key, idPedidoEnvio, phoneNumber, bandeira, sq, smsId, validationCode, tsSend, tsReceive}
}

func SetupRedisPool() (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in SetupRedisPool(): ", r)
		}
	}()
	var redisConnectionString string
	var redisPoolSize int
	var redisDialTimeout int

	if util.AppCfg.RedisOptions.RedisConnectionString == "" {
		redisConnectionString = DefRedisConnectionString
	} else {
		redisConnectionString = util.AppCfg.RedisOptions.RedisConnectionString
	}

	if util.AppCfg.RedisOptions.RedisPoolSize == 0 {
		redisPoolSize = DefRedisPoolSize
	} else {
		redisPoolSize = util.AppCfg.RedisOptions.RedisPoolSize
	}

	if util.AppCfg.RedisOptions.RedisDialTimeout == 0 {
		redisDialTimeout = DefRedisDialTimeout
	} else {
		redisDialTimeout = util.AppCfg.RedisOptions.RedisDialTimeout
	}

	redisByPass, redisClient, errRedis = ConnectToRedis(redisConnectionString, redisPoolSize, false, redisDialTimeout)
	if errRedis != nil {
		util.LogConsole(errRedis.Error())
		util.LogConsole(fmt.Sprintf("SetupRedisPool: server: %s; poolSize: %d; timeout: %d", redisConnectionString, redisPoolSize, redisDialTimeout))
	}
	return redisByPass, errRedis
}

func NextIdPedido() (result string, err error) {
	err = redisClient.Do(radix.Cmd(&result, "INCR", "sms:sq:global"))
	return result, err
}

func NextSQField(key *string, incFieldName string) (result string, err error) {
	err = redisClient.Do(radix.Cmd(&result, "HINCRBY", *key, incFieldName, "1"))
	return result, err
}

func NextSQKey(sqName string) (result string, err error) {
	err = redisClient.Do(radix.Cmd(&result, "INCR", sqName))
	return result, err
}

func nextBilBandeira(bandeira string) (result string, err error) {
	yearMonth := time.Now().Format("06:01")
	sqName := fmt.Sprintf("sms:bil:%s:%s", yearMonth, bandeira)
	return NextSQKey(sqName)
}

func getRequestKey(phoneNumber *string, bandeira *string) string {
	return fmt.Sprintf("sms:rq:%s:%s", *bandeira, *phoneNumber)
}

func DiscardRequestFields(phoneNumber *string, bandeira *string) {
	key := getRequestKey(phoneNumber, bandeira)
	pipe := radix.Pipeline(
		radix.Cmd(nil, "HDEL", key, "idp"),
		radix.Cmd(nil, "HDEL", key, "si"),
		radix.Cmd(nil, "HDEL", key, "sq"),
		radix.Cmd(nil, "HDEL", key, "tsnd"),
	)
	redisClient.Do(pipe)
}

func AccountSMS(bandeira string) string {
	sq, err := nextBilBandeira(bandeira)
	if err == nil {
		util.LogD(fmt.Sprintf("$m$:%s:%s", bandeira, sq))
	} else {
		util.LogE(fmt.Sprintf("$m$error:%s:%s", bandeira, err.Error()))
	}
	return sq
}

func updateLastRequestTry(key *string) error {
	ts := time.Now().Format(time.RFC3339)
	return redisClient.Do(radix.Cmd(nil, "HMSET", *key, "tcts", ts))
}

func readLastRequestTry(key *string) (*time.Time, error) {
	var result []string
	err := redisClient.Do(radix.Cmd(&result, "HMGET", *key, "tcts"))
	if (err == nil) && (result[0] != "") {
		timeRead, err1 := time.Parse(time.RFC3339, result[0])
		return &timeRead, err1
	}
	return nil, err
}

func nextRequestTrycCount(key *string) (int, error) {
	stc, err := NextSQField(key, "tc")
	if (err == nil) && (stc != "") {
		return strconv.Atoi(stc)
	}
	return 0, err
}

func resetTryCount(key *string) {
	pipe := radix.Pipeline(
		radix.Cmd(nil, "HDEL", *key, "tc"),
		radix.Cmd(nil, "HDEL", *key, "tcts"),
	)
	redisClient.Do(pipe)
}

func canRequest(key *string) (bool, error) {
	var lastRequestTime *time.Time
	var err error

	lastRequestTime, err = readLastRequestTry(key)
	if lastRequestTime == nil {
		lastRequestTime = new(time.Time)
		//*lastRequestTime = time.Na//Now()
	}
	interval := time.Now().Sub(*lastRequestTime)
	secondsElapsed := int(math.Round(interval.Seconds()))
	if secondsElapsed < util.DefaultResendWaitSecondsBeforeTriesLimitReached {
		return false, errors.New("Tente novamente em 1 minuto")
	}

	tryCount, err := nextRequestTrycCount(key)
	if err == nil {
		triesLimitReached := tryCount >= util.AppCfg.SmsOptions.MaxSmsRequestsPerPhone
		if triesLimitReached {
			interval = time.Since(*lastRequestTime)
			minutesElapsed := int(math.Round(interval.Minutes()))
			waitIntervalReached := minutesElapsed > util.DefaultResendWaitSecondsAfterTriesLimitReached
			if waitIntervalReached {
				_ = updateLastRequestTry(key)
				resetTryCount(key)
				return true, nil
			} else {
				var msg string
				minutesToWait := util.DefaultResendWaitSecondsAfterTriesLimitReached - minutesElapsed
				if (minutesToWait) <= 1 {
					msg = "1 minuto"
				} else {
					msg = fmt.Sprintf("%d minutos", minutesToWait)
				}
				return false, errors.New("Número máximo de tentativas atingido. Tente novamente em " + msg)
			}
		} else {
			_ = updateLastRequestTry(key)
			return true, nil
		}
	}
	return false, err
}

func logRequest(requestData RequestData) {
	jsonArray := make([]map[string]interface{}, 0, 0)
	entity := make(map[string]interface{})

	contentData := make(map[string]interface{})
	contentData["id"] = requestData.IdPedidoEnvio
	contentData["bandeira_id"] = requestData.Bandeira
	contentData["telefone"] = requestData.PhoneNumber
	contentData["data_hora_requisicao"] = requestData.TimestampSend
	contentData["identificador_sms"] = requestData.SmsId

	entity["entity"] = "historico_envio_sms"
	entity["content"] = contentData
	jsonArray = append(jsonArray, entity)
	jsonBytes, _ := json.Marshal(jsonArray)

	var err error = nil
	var resp *fasthttp.Response
	defer func() {
		if (err != nil) || (resp.StatusCode() != http.StatusOK) {
			writeRequestLogFailed(&requestData)
		}
	}()

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(util.AppCfg.LogMachine + logServiceSaveMethod)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	jsonString := string(jsonBytes)
	req.SetBodyString(jsonString)
	resp = fasthttp.AcquireResponse()
	client := &fasthttp.Client{}
	err = client.Do(req, resp)
}

func writeRequestLogFailed(requestData *RequestData) {
	timestampSend, _ := time.Parse(time.RFC3339, requestData.TimestampSend)
	anoMes := timestampSend.Format("06:01")
	key := fmt.Sprintf("sms:logrq:%s:%s:%s", anoMes, requestData.Bandeira, requestData.Sq)
	err := redisClient.Do(radix.Cmd(nil, "HMSET", key, "idp", requestData.IdPedidoEnvio, "pn", requestData.PhoneNumber, "si", requestData.SmsId, "tsnd", requestData.TimestampSend))
	if err != nil {
		util.LogConsole(err.Error())
	}
}

func logResponse(responseData ResponseData) {
	jsonArray := make([]map[string]interface{}, 0, 0)
	entity := make(map[string]interface{})

	contentData := make(map[string]interface{})
	contentData["id"] = responseData.IdPedidoEnvio
	codigoValidacao, _ := strconv.Atoi(responseData.ValidationCode)
	contentData["codigo_validacao"] = codigoValidacao
	contentData["data_hora_confirmacao"] = responseData.TimestampReceive

	entity["entity"] = "historico_confirmacao_sms"
	entity["content"] = contentData
	jsonArray = append(jsonArray, entity)
	jsonBytes, _ := json.Marshal(jsonArray)

	var err error = nil
	var resp *fasthttp.Response
	defer func() {
		if (err != nil) || (resp.StatusCode() != http.StatusOK) {
			writeResponseLogFailed(&responseData)
		}
	}()

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(util.AppCfg.LogMachine + logServiceSaveMethod)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.SetBodyString(string(jsonBytes))
	resp = fasthttp.AcquireResponse()
	client := &fasthttp.Client{}
	err = client.Do(req, resp)
}

func writeResponseLogFailed(responseData *ResponseData) {
	timestampSend, _ := time.Parse(time.RFC3339, responseData.TimestampSend)
	anoMes := timestampSend.Format("06:01")
	key := fmt.Sprintf("sms:logrs:%s:%s:%s", anoMes, responseData.Bandeira, responseData.Sq)
	err := redisClient.Do(radix.Cmd(nil, "HMSET", key, "idp", responseData.IdPedidoEnvio, "cv", responseData.ValidationCode, "trcv", responseData.TimestampReceive))
	if err != nil {
		util.LogConsole(err.Error())
	}
}

func WriteRequest(reqData RequestData) (RequestData, error) {
	key := getRequestKey(&reqData.PhoneNumber, &reqData.Bandeira)
	//Se reqData.Sq é vazio, está inserindo, senão, está atualizando
	//Só incrementa quando insere
	isInserting := reqData.Sq == ""
	if isInserting {
		canDoRequest, err := canRequest(&key)
		if !canDoRequest {
			return reqData, err
		}
		sRequestCount, _ := NextSQField(&key, "total")
		util.LogD("requestCount: " + sRequestCount)
	}
	idPedido := reqData.IdPedidoEnvio
	if idPedido == "" {
		idPedido, _ = NextIdPedido()
	}
	ts := time.Now().Format(time.RFC3339)
	resultReqData := NewRequestData(key, idPedido, reqData.PhoneNumber, reqData.Bandeira, reqData.Sq, reqData.SmsId, ts)
	err := redisClient.Do(radix.Cmd(nil, "HMSET", key, "idp", resultReqData.IdPedidoEnvio, "sq", resultReqData.Sq, "si", resultReqData.SmsId, "tsnd", resultReqData.TimestampSend))
	if err == nil {
		//Só tem as informações completas quando atualiza e só atualiza quando de fato solicitou um envio de SMS
		if !isInserting {
			go logRequest(resultReqData)
		}
		sIdTTL := fmt.Sprintf("%d", requestTTL)
		err = redisClient.Do(radix.Cmd(nil, "EXPIRE", key, sIdTTL))
		if err != nil {
			return resultReqData, errors.New("Não foi possível armazenar a validade do pedido")
		}
	} else {
		return resultReqData, errors.New("Não foi possível armazenar o pedido")
	}

	return resultReqData, err
}

func ReadRequest(phoneNumber *string, bandeira *string) (*RequestData, error) {
	var result []string
	key := getRequestKey(phoneNumber, bandeira)
	err := redisClient.Do(radix.Cmd(&result, "HMGET", key, "idp", "sq", "si", "tsnd"))
	if err == nil {
		resultRequestData := NewRequestData(key, result[0], *phoneNumber, *bandeira, result[1], result[2], result[3])
		return &resultRequestData, err
	} else {
		return nil, err
	}
}

func WriteResponse(responseData *ResponseData) (*ResponseData, error) {
	key := fmt.Sprintf("sms:rs:%s:%s:%s", time.Now().Format("06:01"), responseData.Bandeira, responseData.Sq)
	trcv := time.Now().Format(time.RFC3339)
	resultResponseData := NewResponseData(key, responseData.IdPedidoEnvio, responseData.PhoneNumber, responseData.Bandeira, responseData.Sq, responseData.SmsId, responseData.ValidationCode, responseData.TimestampSend, trcv)
	err := redisClient.Do(radix.Cmd(nil, "HMSET", key, "idp", responseData.IdPedidoEnvio, "pn", responseData.PhoneNumber, "si", responseData.SmsId, "vc", responseData.ValidationCode, "tsnd", responseData.TimestampSend, "trcv", trcv))
	go logResponse(resultResponseData)
	if err == nil {
		reqKey := getRequestKey(&responseData.PhoneNumber, &responseData.Bandeira)
		resetTryCount(&reqKey)
		//token para localizar os dados na fase de cadastro no php. O php chama findToken para obter o telefone confirmado
		err = writeTempToken(responseData.SmsId, responseData.PhoneNumber, responseData.ValidationCode)
		if err == nil {
			return &resultResponseData, err
		} else {
			return nil, err
		}
	}
	return nil, err
}

func WriteFail(responseData *ResponseData) (*ResponseData, error) {
	key := fmt.Sprintf("sms:rs:%s:%s:%s", time.Now().Format("06:01"), responseData.Bandeira, responseData.Sq)
	err := redisClient.Do(radix.Cmd(nil, "HMSET", key, "idp", responseData.IdPedidoEnvio, "pn", responseData.PhoneNumber, "si", responseData.SmsId, "tsnd", responseData.TimestampSend))
	if err == nil {
		reqKey := getRequestKey(&responseData.PhoneNumber, &responseData.Bandeira)
		resetTryCount(&reqKey)
		err = writeTempToken(responseData.SmsId, responseData.PhoneNumber, responseData.ValidationCode)
		if err == nil {
			resultResponseData := NewResponseData(key, responseData.IdPedidoEnvio, responseData.PhoneNumber, responseData.Bandeira, responseData.Sq, responseData.SmsId, responseData.ValidationCode, responseData.TimestampSend, responseData.TimestampReceive)
			return &resultResponseData, err
		} else {
			return nil, err
		}
	}
	return nil, err
}

func MovePossibleFailedRequest(phoneNumber *string, bandeira *string) error {
	reqData, err := ReadRequest(phoneNumber, bandeira)
	if (err == nil) && (reqData != nil) && (reqData.SmsId != "") {
		respData := NewResponseData("", reqData.IdPedidoEnvio, *phoneNumber, *bandeira, reqData.Sq, reqData.SmsId, "", reqData.TimestampSend, "")
		_, err = WriteFail(&respData)
	}
	return err
}

func writeTempToken(smsId string, phoneNumber string, validationCode string) error {
	sTempTokenTTL := fmt.Sprintf("%d", tempTokenTTL)
	pipe := radix.Pipeline(
		radix.Cmd(nil, "HMSET", smsId, "pn", phoneNumber, "vc", validationCode),
		radix.Cmd(nil, "EXPIRE", smsId, sTempTokenTTL),
	)
	return redisClient.Do(pipe)
}

func FindTempToken(smsId string) (phoneNumber string, validationCode string, err error) {
	var result []string
	err = redisClient.Do(radix.Cmd(&result, "HMGET", smsId, "pn", "vc"))
	phoneNumber = result[0]
	validationCode = result[1]
	return phoneNumber, validationCode, err
}

// ConnectToRedis Open connection to Redis
func ConnectToRedis(redisEndpoint string, redisPoolSize int, redisBypass bool, dialTimeout int) (bool, *radix.Pool, error) {
	if redisBypass {
		log.Println("Skipping redis connection: redis bypass is on.")
		return redisBypass, nil, nil
	}

	customConnFunc := func(network, addr string) (radix.Conn, error) {
		return radix.Dial(network, addr, radix.DialTimeout(time.Duration(dialTimeout)*time.Second))
	}

	redisClient, errRedis := radix.NewPool("tcp", redisEndpoint, redisPoolSize, radix.PoolConnFunc(customConnFunc))

	if errRedis != nil {
		log.Println("Error connecting to redis: turning on redis bypass. -", errRedis)
		redisBypass = true
		return redisBypass, nil, errRedis
	}

	log.Println("Connection to redis performed successfuly")

	return redisBypass, redisClient, errRedis
}

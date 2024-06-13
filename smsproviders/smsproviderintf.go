package smsproviders

const (
	Success   = true
	SuccessCode = 0
	NoSuccess = false
)

type SmsResult struct {
	IsSuccess	bool
	Code		int
	Msg			string
	Data		interface{}
}

func NewSmsResult(isSuccess bool, code int, msg string, data interface{}) *SmsResult {
	return &SmsResult{IsSuccess: isSuccess, Code: code, Msg: msg, Data: data}
}

type SmsProviderIntf interface {
	ProviderName() string

	SendVerificationRequest(phoneNumber string, content string, hashCode string) (result SmsResult)
	CheckSendVerificationResponse(content []byte) (result SmsResult)

	/*SendMessageRequest(phoneNumber string, content string, hashCode string) (result SmsResult)
	CheckSendMessageResponse(content []byte) (result SmsResult)*/

	VerifyRequest(phoneNumber string, sentCode string, receivedCode string) (result SmsResult)
	CheckVerifyResponse(content []byte) (result SmsResult)

}

package util

const (
	DefaultHttpPort		= 80
	DefaultConfigPath   = "./etc/"
	DefaultConfigFile   = "sms.conf"
	DefaultLogPath      = "./var/log/microservices/"
	DefaultLogFile      = "sms.log"
)

var AppCfg Config
//Valores do config sobrescrevem estes valores default
var DefaultMaxSmsRequestsPerPhone int = 5
var DefaultResendWaitSecondsBeforeTriesLimitReached int = 45 //45 segundos - Só envia novo SMS com, pelo menos, %d segundos de intervalo, que é o intervalo da validação
var DefaultResendWaitSecondsAfterTriesLimitReached int = 2   //minutos - Depois de %d tentativas, exige que se aguarde %d minutos para iniciar um novo ciclo de tentativas

func NewConfig(defRedisConnectionString string, defRedisPoolSize int, defRedisDialTimeout int, defaultPort int, defaultMaxSmsRequestsPerPhone int, resendWaitSecondsAfterTriesLimitReached int) Config {
	return Config{DefaultLogPath + DefaultLogFile,
		"D,I,W,E",
		true,
		"http://elb-kdev-microservices.gaudium.lan/api/logmachine",
		"",
		"",
		Redis{defRedisConnectionString, defRedisPoolSize, defRedisDialTimeout},
		Network{defaultPort},
		Sms{defaultMaxSmsRequestsPerPhone, resendWaitSecondsAfterTriesLimitReached}}
}

type Config struct {
	LogFileName        string
	LogOptions         string
	LogEnabled         bool
	LogMachine         string
	ChangeProviderKey  string
	ChangeLogStatusKey string
	RedisOptions       Redis
	NetworkOptions     Network
	SmsOptions         Sms
}

type Redis struct {
	RedisConnectionString string
	RedisPoolSize         int
	RedisDialTimeout      int
}

type Network struct {
	ListeningPort int
}

type Sms struct {
	SmsSecureRequestIntervalInMinutes int
	MaxSmsRequestsPerPhone            int
}

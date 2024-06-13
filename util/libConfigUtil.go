package util

import (
	"fmt"
	toml "github.com/pelletier/go-toml"
	"log"
	"os"
	"reflect"
)

func InitLog(logFileName string) *os.File {
	f, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	log.Println("Starting sms server")
	return f
}

func loadConfigSections(conf *toml.Tree, configStruct interface{}) {
	t := reflect.TypeOf(configStruct)
    v := reflect.ValueOf(configStruct)
	if (t.Kind() == reflect.Ptr) {
		typeOfS := v.Elem().Type()
		sectionName := t.Elem().Name()
		for i := 0; i < v.Elem().NumField(); i++ {
			if v.Elem().Field(i).CanInterface() {
				if v.Elem().Field(i).Kind() == reflect.Struct {
					if v.Elem().Field(i).CanInterface() && v.Elem().Field(i).CanAddr() {
						loadConfigSections(conf, v.Elem().Field(i).Addr().Interface())
					}
				} else {
					kValue := conf.Get(sectionName + "." + typeOfS.Field(i).Name)
					if kValue != nil {
						kObjValue := reflect.ValueOf(kValue)
						if (kObjValue.IsValid()) {
							v.Elem().Field(i).Set(kObjValue.Convert(typeOfS.Field(i).Type))
						}
					}
				}
			}
		}
	}
}

func LoadConfig(configFileName string, configStruct interface{}) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in LoadConfig: ", r)
		}
	}()
	conf, err := toml.LoadFile(configFileName)
	if err == nil {
		loadConfigSections(conf, configStruct)
	} else {
		LogE("LoadConfig.Error: " + err.Error())
	}
}

func PrintConfig(configStruct interface{}, printToLog bool) {

	v := reflect.ValueOf(configStruct)
	typeOfS := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).CanInterface() {
			value := fmt.Sprintf("%s = %v", v.Type().Name()+"."+typeOfS.Field(i).Name, v.Field(i).Interface())
			if printToLog {
				LogI(value)
			} else {
				fmt.Println(value)
			}
		}
	}
}

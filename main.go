package main

import (
	"log"
	"mydnns/service/myrpc"

	"github.com/spf13/viper"
)

// initConfig 读取配置文件
func initConfig() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("init config fail", err)
		return
	}

}
func main() {
	initConfig()
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
	// 启动服务
	myrpc.StartServer()
}

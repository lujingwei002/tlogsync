package config

import (
	"log"

	"encoding/json"

	"gopkg.in/ini.v1"
)

var Ini struct {
	Basic struct {
		Debug bool `ini:"debug"`
	} `ini:"basic"`
	MySql struct {
		Ip       string `ini:"ip"`
		Port     int    `ini:"port"`
		User     string `ini:"user"`
		Password string `ini:"password"`
		Db       string `ini:"db"`
		Charset  string `ini:"charset"`
	} `ini:"mysql"`

	Tlog struct {
		Dir             string `ini:"dir"`
		BackupDir       string `ini:"backupdir"`
		BatchWrite      int    `ini:"batchwrite"`
		SyncTime        int64  `ini:"synctime"`
		Listen          string `ini:"listen"`
		LogXml          string `ini:"logxml"`
		AutoCreateTable bool   `ini:"autocreatetable"`
		AutoAddColumn   bool   `ini:"autoaddcolumn"`
	} `ini:"tlog"`
}

func init() {
	err := ini.MapTo(&Ini, "config.ini")
	if err != nil {
		panic(err)
	}
	str, err := json.MarshalIndent(Ini, "", "\t")
	if err != nil {
		panic(err)
	}
	log.Printf("[config] %s\n", str)
}

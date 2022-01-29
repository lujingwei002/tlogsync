package db

import (
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/shark/minigame-tlogsync/config"
)

var db *sqlx.DB

func init() {
	addr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s",
		config.Ini.MySql.User, config.Ini.MySql.Password, config.Ini.MySql.Ip, config.Ini.MySql.Port, config.Ini.MySql.Db, config.Ini.MySql.Charset)
	var err error
	db, err = sqlx.Open("mysql", addr)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	log.Printf("连接数据库成功 %+v\n", addr)
	if err := loadModelXml(config.Ini.Tlog.LogXml); err != nil {
		panic(err)
	}
	syncDatabase2()
	go forkSyncDatabase()
}

func forkSyncDatabase() {
	for {
		time.Sleep(10 * 24 * time.Hour)
		syncDatabase2()
	}
}

func syncDatabase2() error {
	//当月
	monthTime := time.Now()
	month := monthTime.Year()*100 + int(monthTime.Month())
	//log.Println("建表", month)
	syncDatabase(month)
	//下月
	nextMonthTime := time.Now().AddDate(0, 1, 0)
	nextMonth := nextMonthTime.Year()*100 + int(nextMonthTime.Month())
	//log.Println("建表", nextMonth)
	syncDatabase(nextMonth)
	return nil
}

func syncDatabase(suffix int) error {
	if config.Ini.Tlog.AutoCreateTable {
		autoCreateTable(suffix)
	}
	if config.Ini.Tlog.AutoAddColumn {
		autoAddColumn(suffix)
	}
	return nil
}

func logtime2Month(t int64) int {
	now := time.Unix(t, 0)
	month := now.Year()*100 + int(now.Month())
	return month
}

func tableIsExits(tableName string) bool {
	_, err := db.Exec(fmt.Sprintf("desc %s", tableName))
	if err == nil {
		return true
	}
	return false
}

func Common_Insert(tlogModel *TlogModel, typ string, rows [][]string, logtime int64) error {
	now := time.Now().Unix()
	month := logtime2Month(logtime)
	tableName := strings.ToLower(typ)
	if tlogModel.Sharding == "month" {
		tableName = fmt.Sprintf("%s_%d", strings.ToLower(typ), month)
	}
	sql := fmt.Sprintf("INSERT INTO %s %s VALUES ", tableName, tlogModel.fieldSql)
	args0 := rows[0]
	oneValueArr := make([]string, 0)
	for i := 1; i < len(args0)+2; i++ {
		oneValueArr = append(oneValueArr, "?")
	}
	valueStr := strings.Join(oneValueArr, ",")
	valueStr = "(" + valueStr + ")"
	valueArr := make([]string, 0)
	for i := 0; i < len(rows); i++ {
		valueArr = append(valueArr, valueStr)
	}
	sql = fmt.Sprintf("%s%s", sql, strings.Join(valueArr, ","))
	args := make([]interface{}, 0)
	for _, row := range rows {
		args = append(args, row[1]) //version
		args = append(args, row[2]) //logtime
		args = append(args, now)    //createtime
		args = append(args, now)    //updatetime
		for _, v := range row[3:] {
			args = append(args, v)
		}
	}
	if config.Ini.Basic.Debug {
		log.Println(sql, args)
	}
	_, err := db.Exec(sql, args...)
	if err != nil {
		log.Printf("db.Common_Insert err %+v\n", err)
		return err
	}
	return nil
}

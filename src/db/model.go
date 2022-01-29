package db

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/shark/minigame-tlogsync/config"
)

var tlogDict map[string]*TlogModel
var tlogVerDict map[string]*TlogModel
var tlogArr []*TlogModel

type TlogModel struct {
	fieldSql  string
	fieldDict map[string]*TlogField
	VerName   string
	Version   int          `xml:"version,attr"`
	FieldArr  []*TlogField `xml:"field"`
	Name      string       `xml:"name,attr"`
	Comment   string       `xml:"comment,attr"`
	Sharding  string       `xml:"sharding,attr"`
}

type TlogField struct {
	Name    string `xml:"name,attr"`
	Type    string `xml:"type,attr"`
	Comment string `xml:"comment,attr"`
	Index   bool   `xml:"index,attr"`
}

type tlogXml struct {
	TlogArr []*TlogModel `xml:"tlog"`
}

func loadModelXml(filename string) error {
	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var x tlogXml
	if err := xml.Unmarshal(bs, &x); err != nil {
		return err
	}

	for _, tlogModel := range x.TlogArr {
		tlogModel.FieldArr = append([]*TlogField{&TlogField{
			Name:    "version",
			Type:    "int",
			Comment: "版本",
		}, &TlogField{
			Name:    "logtime",
			Type:    "int",
			Comment: "日志时间",
			Index:   true,
		}, &TlogField{
			Name:    "createtime",
			Type:    "int",
			Comment: "创建时间",
		}, &TlogField{
			Name:    "updatetime",
			Type:    "int",
			Comment: "更新时间",
		}}, tlogModel.FieldArr[0:]...)
		tlogModel.fieldDict = make(map[string]*TlogField)
		for _, field := range tlogModel.FieldArr {
			tlogModel.fieldDict[field.Name] = field
		}
		tlogModel.fieldSql = tlogModel.formFieldSql()
		tlogModel.VerName = fmt.Sprintf("%sv%d", tlogModel.Name, tlogModel.Version)
		if config.Ini.Basic.Debug {
			log.Println(tlogModel.formCreateTableSQL())
			for _, field := range tlogModel.FieldArr {
				if field.Index {
					log.Println(field.formAddIndexSql(tlogModel.Name))
				}
			}
		}
	}
	tlogDict = make(map[string]*TlogModel)
	tlogVerDict = make(map[string]*TlogModel)
	tlogArr = make([]*TlogModel, 0)
	for _, tlogModel := range x.TlogArr {
		tlogVerDict[tlogModel.VerName] = tlogModel
		if lastTlogModel, ok := tlogDict[tlogModel.Name]; !ok || (ok && tlogModel.Version > lastTlogModel.Version) {
			tlogDict[tlogModel.Name] = tlogModel
		}
		tlogArr = append(tlogArr, tlogModel)
	}
	//log.Printf("afasf %+v\n", tlogDict["gatestat"])
	//log.Printf("afasf %+v\n", tlogVerDict["gatestatv2"].fieldSql)
	return nil
}

func (tlog *TlogModel) formFieldSql() string {
	fieldNameArr := make([]string, 0)
	for _, field := range tlog.FieldArr {
		fieldNameArr = append(fieldNameArr, field.Name)
	}
	sql := "(" + strings.Join(fieldNameArr, ",") + ")"
	return sql
}

func (tlog *TlogModel) formCreateTableSQL() string {
	sql := fmt.Sprintf("CREATE TABLE `%s` (\n", tlog.Name)
	sql = sql + "\t`id` bigint(11) AUTO_INCREMENT COMMENT 'id',\n"
	for _, field := range tlog.FieldArr {
		sql = sql + fmt.Sprintf("\t%s,\n", field.formColumnSql())
	}
	sql = sql + "\tPRIMARY KEY (`id`)\n"
	sql = sql + fmt.Sprintf(") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=COMPACT COMMENT='%s';", tlog.Comment)
	return sql
}

func (f *TlogField) formAddColumnSql(tableName string) string {
	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, f.formColumnSql())
	return sql
}

func (f *TlogField) formAddIndexSql(tableName string) string {
	sql := fmt.Sprintf("ALTER TABLE %s ADD INDEX i_%s(`%s`)", tableName, f.Name, f.Name)
	return sql
}

func (f *TlogField) formDropColumnSql(tableName string) string {
	sql := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, f.Name)
	return sql
}

func (f *TlogField) formColumnSql() string {
	if strings.Index(f.Type, "varchar") == 0 {
		return fmt.Sprintf("`%s` %s NOT NULL DEFAULT '' COMMENT '%s'", f.Name, f.Type, f.Comment)
	} else {
		return fmt.Sprintf("`%s` %s NOT NULL DEFAULT '0' COMMENT '%s'", f.Name, f.Type, f.Comment)
	}
	return ""
}

func GetTlogModel(typ string) *TlogModel {
	tlog, ok := tlogVerDict[typ]
	if !ok {
		return nil
	}
	return tlog
}

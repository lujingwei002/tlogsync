package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

type fieldSchema struct {
	Field   string         `db:"Field"`
	Type    string         `db:"Type"`
	Null    string         `db:"Null"`
	Key     string         `db:"Key"`
	Default sql.NullString `db:"Default"`
	Extra   string         `db:"Extra"`
}

type tableSchema struct {
	fieldArr  []*fieldSchema
	fieldDict map[string]*fieldSchema
}

type indexSchema struct {
	Table         string         `db:"Table"`
	Non_unique    int            `db:"Non_unique"`
	KeyName       string         `db:"Key_name"`
	Seq_in_index  int            `db:"Seq_in_index"`
	ColumnName    string         `db:"Column_name"`
	Collation     string         `db:"Collation"`
	Cardinality   string         `db:"Cardinality"`
	Sub_part      sql.NullString `db:"Sub_part"`
	Packed        sql.NullString `db:"Packed"`
	Null          sql.NullString `db:"Null"`
	Index_type    string         `db:"Index_type"`
	Comment       string         `db:"Comment"`
	Index_comment string         `db:"Index_comment"`
	Visible       string         `db:"Visible"`
}

type tableIndexSchema struct {
	indexArr  []*indexSchema
	indexDict map[string]*indexSchema
}

func getTableSchema(tableName string) (*tableSchema, error) {
	fieldArr := make([]*fieldSchema, 0)
	err := db.Select(&fieldArr, "desc "+tableName)
	if err != nil {
		return nil, err
	}
	fieldDict := make(map[string]*fieldSchema)
	for _, field := range fieldArr {
		fieldDict[field.Field] = field
	}
	schema := &tableSchema{
		fieldArr:  fieldArr,
		fieldDict: fieldDict,
	}
	return schema, nil
}

func getTableIndexSchema(tableName string) (*tableIndexSchema, error) {
	indexArr := make([]*indexSchema, 0)
	err := db.Select(&indexArr, "SHOW INDEX FROM "+tableName)
	if err != nil {
		return nil, err
	}
	indexDict := make(map[string]*indexSchema)
	for _, index := range indexArr {
		indexDict[index.KeyName] = index
	}
	schema := &tableIndexSchema{
		indexArr:  indexArr,
		indexDict: indexDict,
	}
	return schema, nil
}

func autoCreateTable(suffix int) error {
	for _, tlogModel := range tlogDict {
		tableName := tlogModel.Name
		if tlogModel.Sharding == "month" {
			tableName = fmt.Sprintf("%s_%d", tlogModel.Name, suffix)
		}
		log.Println("检查创建表", tableName)
		if tableIsExits(tableName) {
			continue
		}
		//创建表
		log.Println("创建表", tableName)
		sql := tlogModel.formCreateTableSQL()
		sql = strings.Replace(sql, tlogModel.Name, tableName, 1)
		log.Println(sql)
		_, err := db.Exec(sql)
		if err != nil {
			log.Printf("创建表失败, 原因=%s\n", err.Error())
		}
		for _, field := range tlogModel.FieldArr {
			if !field.Index {
				continue
			}
			if _, err := db.Exec(field.formAddIndexSql(tableName)); err != nil {
				log.Printf("添加索引失败, 原因=%s\n", err.Error())
			}
		}
	}
	return nil
}

func autoAddColumn(suffix int) error {
	for _, tlogModel := range tlogDict {
		tableName := tlogModel.Name
		if tlogModel.Sharding == "month" {
			tableName = fmt.Sprintf("%s_%d", tlogModel.Name, suffix)
		}
		log.Println("检查增加列", tableName)
		schema, err := getTableSchema(tableName)
		if err != nil {
			log.Printf("获取表结构失败, 原因=%s\n", err.Error())
			continue
		}
		//检查是否有新字段
		for _, field := range tlogModel.FieldArr {
			if _, ok := schema.fieldDict[field.Name]; !ok {
				sql := field.formAddColumnSql(tableName)
				log.Println(sql)
				_, err := db.Exec(sql)
				if err != nil {
					log.Printf("修改表失败, 原因=%s\n", err.Error())
				}
			}
		}

		indexSchema, err := getTableIndexSchema(tableName)
		if err != nil {
			log.Printf("获取表索引失败, 原因=%s\n", err.Error())
			continue
		}
		//检查是否有索引
		for _, field := range tlogModel.FieldArr {
			if !field.Index {
				continue
			}
			if _, ok := indexSchema.indexDict["i_"+field.Name]; !ok {
				sql := field.formAddIndexSql(tableName)
				log.Println(sql)
				_, err := db.Exec(sql)
				if err != nil {
					log.Printf("添加索引失败, 原因=%s\n", err.Error())
				}
			}
		}
	}
	return nil
}

func autoDropColumn(suffix int) error {
	for _, tlogModel := range tlogDict {
		tableName := tlogModel.Name
		if tlogModel.Sharding == "month" {
			tableName = fmt.Sprintf("%s_%d", tlogModel.Name, suffix)
		}
		log.Println("检查删除列", tableName)
		schema, err := getTableSchema(tableName)
		if err != nil {
			log.Printf("获取表结构失败, 原因=%s\n", err.Error())
			continue
		}
		//检查是否需要删除字段
		for _, field := range schema.fieldArr {
			if field.Field == "id" {
				continue
			}
			if field.Field == "version" {
				continue
			}
			if _, ok := tlogModel.fieldDict[field.Field]; !ok {
				sql := fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN %s", tableName, field.Field)
				log.Println(sql)
				_, err := db.Exec(sql)
				if err != nil {
					log.Printf("修改表失败, 原因=%s\n", err.Error())
				}
			}
		}
	}
	return nil
}

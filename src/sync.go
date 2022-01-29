package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/shark/minigame-tlogsync/config"
	"github.com/shark/minigame-tlogsync/db"
)

var errLogFormat = errors.New("log format")

type TlogHandler func(logtime int64, typ string, args [][]string) error

type Cache struct {
	lines     []string
	logtime   int64
	version   int32
	tlogModel *db.TlogModel
}

func (c *Cache) len() int {
	return len(c.lines)
}

func (c *Cache) push(line string) {
	c.lines = append(c.lines, line)
}

type LogSync struct {
	watch    *fsnotify.Watcher
	fileChan chan string
	logChan  chan string
	logCache map[string]*Cache
	listener net.Listener

	chDie         chan bool
	shutDownGroup sync.WaitGroup
}

func newLogSync() (*LogSync, error) {
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	sync := &LogSync{
		watch:    watch,
		fileChan: make(chan string, 1),
		logChan:  make(chan string, 1),
		logCache: make(map[string]*Cache),
		chDie:    make(chan bool),
	}
	return sync, nil
}

func (s *LogSync) run() {
	//同步目录里的文件
	if err := s.syncDir(config.Ini.Tlog.Dir); err != nil {
		log.Fatalln(err)
	}
	go s.forkSync()
	//监控文件
	go s.watchTlogDir()
	//开启server
	go s.listenAndServer()
}

func (s *LogSync) shutDown() {
	log.Println("shutdown1")
	close(s.chDie)
	s.shutDownGroup.Wait()
	s.flushAllCache()
	log.Println("shutdown2")
}

//同步所有文件
func (s *LogSync) syncDirFunc(path string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}
	s.syncFile(path)
	return nil
}

//同步所有文件
func (s *LogSync) syncDir(dir string) error {
	if err := filepath.Walk(dir, s.syncDirFunc); err != nil {
		return err
	}
	return nil
}

//检查是否日志文件
func (s *LogSync) checkTlogFile(path string) bool {
	name := filepath.Base(path)
	ext := filepath.Ext(path)
	if ext != ".log" {
		return false
	}
	//删除扩展名
	name = strings.Replace(name, ext, "", 1)
	args := strings.Split(name, "_")
	if len(args) <= 2 {
		return false
	}
	if args[len(args)-2] != "tlog" {
		return false
	}
	if _, err := strconv.ParseInt(args[len(args)-1], 10, 64); err != nil {
		return false
	}
	return true
}

//同步单个文件
func (s *LogSync) syncFile(path string) error {
	if !s.checkTlogFile(path) {
		log.Println("无效文件", path)
		return nil
	}
	log.Println("同步文件", path)
	file, err := os.Open(path)
	if nil != err {
		return err
	}
	buff := bufio.NewReader(file)
	for {
		line, err := buff.ReadString('\n')
		if err == io.EOF {
			break
		}
		//删掉换行
		line = strings.TrimSpace(line)
		s.syncTlog(line)
	}
	//批量写入
	s.flushAllCache()
	file.Close()
	//备份文件
	if err := s.backupFile(path); err != nil {
		return err
	}
	return nil
}

func (s *LogSync) syncTlog(line string) error {
	line = strings.TrimSpace(line)
	//log.Println("读取", line)
	args := strings.Split(line, "|")
	if len(args) <= 0 {
		return nil
	}
	//先加入缓存，一会批量写入
	typ := args[0]
	version := atoi32(args[1])
	logtime := atoi64(args[2])
	tlogModel := db.GetTlogModel(fmt.Sprintf("%sv%d", typ, version))
	if tlogModel == nil {
		log.Printf("过滤日志,请检查xml %s\n", line)
		return nil
	}
	if len(args)-1 != len(tlogModel.FieldArr)-2 {
		log.Printf("日志不符合长度规则 长度要求:%d, %s\n", len(tlogModel.FieldArr)-2, line)
		return nil
	}
	cache, ok := s.logCache[typ]
	if ok && (!isSameMonth(cache.logtime, logtime) || cache.version != version) {
		//跨月的话，立刻刷新
		s.flushCache(typ, cache)
		delete(s.logCache, typ)
	}
	cache, ok = s.logCache[typ]
	if ok {
		cache.push(line)
	} else {
		cache = &Cache{
			lines:     make([]string, 0),
			logtime:   logtime,
			version:   version,
			tlogModel: tlogModel,
		}
		cache.push(line)
		s.logCache[typ] = cache
	}
	if cache.len() >= config.Ini.Tlog.BatchWrite {
		s.flushCache(typ, cache)
		delete(s.logCache, typ)
	}
	return nil
}

//换文件时,批量写入所有日志
func (sync *LogSync) flushAllCache() error {
	log.Println("刷新全部日志")
	for typ, cache := range sync.logCache {
		sync.flushCache(typ, cache)
	}
	sync.logCache = make(map[string]*Cache)
	return nil
}

//批量写入日志
func (s *LogSync) flushCache(typ string, cache *Cache) error {
	//log.Println("刷新日志", typ, s.logCache[typ])
	rows := make([][]string, 0)
	for _, line := range cache.lines {
		args := strings.Split(line, "|")
		if len(args) <= 0 {
			log.Println("无效日志", line)
			return nil
		}
		//log.Println("写入日志", line)
		rows = append(rows, args)
	}
	if err := s.tlogCommon(cache.tlogModel, typ, rows, cache.logtime); err != nil {
		return err
	}
	return nil
}

//备份文件
func (s *LogSync) backupFile(path string) error {
	//return nil
	backupPath := strings.Replace(path, config.Ini.Tlog.Dir, config.Ini.Tlog.BackupDir, 1)
	log.Println("备份文件", path, "=>", backupPath)
	dir := filepath.Dir(backupPath)
	if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0666); err != nil {
			return nil
		}
	}
	if err := os.Rename(path, backupPath); err != nil {
		return nil
	}
	return nil
}

//监听文件变化
func (s *LogSync) forkSync() {
	tick := time.NewTicker(time.Duration(config.Ini.Tlog.SyncTime) * time.Second)
	defer func() {
		log.Println("sync done")
		tick.Stop()
		s.shutDownGroup.Done()
	}()
	s.shutDownGroup.Add(1)
	for {
		select {
		case path := <-s.fileChan:
			{
				s.syncFile(path)
			}
		case line := <-s.logChan:
			{
				s.syncTlog(line)
			}
		case <-tick.C:
			{
				s.flushAllCache()
			}
		case <-s.chDie:
			{
				return
			}
		}
	}
}

func (s *LogSync) tlogCommon(tlogModel *db.TlogModel, typ string, lines [][]string, logtime int64) error {
	if err := db.Common_Insert(tlogModel, typ, lines, logtime); err != nil {
		return err
	}
	return nil
}

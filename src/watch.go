package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/shark/minigame-tlogsync/config"
)

func (s *LogSync) watchTlogDirFunc(path string, info os.FileInfo, err error) error {
	if !info.IsDir() {
		return nil
	}
	//添加要监控的对象，文件夹
	err = s.watch.Add(path)
	if err != nil {
		return err
	}
	log.Println("监控目录", path)
	return nil
}

func (s *LogSync) watchTlogDir() {
	if _, err := os.Stat(config.Ini.Tlog.Dir); err != nil && os.IsNotExist(err) {
		log.Fatalln(err)
	}
	if err := filepath.Walk(config.Ini.Tlog.Dir, s.watchTlogDirFunc); err != nil {
		log.Fatalln(err)
	}
	defer func() {
		log.Println("watch done")
		s.watch.Close()
		s.shutDownGroup.Done()
	}()
	s.shutDownGroup.Add(1)
	for {
		select {
		case ev := <-s.watch.Events:
			{
				//判断事件发生的类型，如下5种
				// Create 创建
				// Write 写入
				// Remove 删除
				// Rename 重命名
				// Chmod 修改权限
				if ev.Op&fsnotify.Create == fsnotify.Create {
					log.Println("创建文件 : ", ev.Name)
					if finfo, err := os.Stat(ev.Name); err == nil {
						if finfo.IsDir() {
							//添加要监控的对象，文件夹
							err = s.watch.Add(ev.Name)
							if err != nil {
								log.Printf("watch dir failed, error=%s\n", err.Error())
							}
							log.Println("监控目录", ev.Name)
						} else {
							s.fileChan <- ev.Name
						}
					}
				}
				if ev.Op&fsnotify.Write == fsnotify.Write {
					log.Println("写入文件 : ", ev.Name)
				}
				if ev.Op&fsnotify.Remove == fsnotify.Remove {
					log.Println("删除文件 : ", ev.Name)
				}
				if ev.Op&fsnotify.Rename == fsnotify.Rename {
					log.Println("重命名文件 : ", ev.Name)
				}
				if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
					log.Println("修改权限 : ", ev.Name)
				}
			}
		case err := <-s.watch.Errors:
			{
				log.Println("error : ", err)
				return
			}
		case <-s.chDie:
			{
				return
			}
		}
	}
}

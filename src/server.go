package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strings"

	"github.com/shark/minigame-tlogsync/config"
)

func (s *LogSync) listenAndServer() {
	if len(config.Ini.Tlog.Listen) <= 0 {
		return
	}
	ln, err := net.Listen("tcp", config.Ini.Tlog.Listen)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("监听tcp", config.Ini.Tlog.Listen)
	defer func() {
		log.Println("listener done")
		s.shutDownGroup.Done()
	}()
	s.shutDownGroup.Add(1)
	s.listener = ln
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			return
		}
		go s.handleConnection(conn)
	}
}

func (s *LogSync) handleConnection(conn net.Conn) {
	log.Println("接受tcp链接")
	buff := bufio.NewReader(conn)
	for {
		line, err := buff.ReadString('\n')
		if err == io.EOF {
			break
		}
		//删掉换行
		line = strings.Replace(line, "\n", "", 1)
		//s.syncTlog(line)
		log.Println(line)
		s.logChan <- line
	}
	log.Println("断开tcp链接")
}

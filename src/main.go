package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/shark/minigame-tlogsync/config"
	_ "github.com/shark/minigame-tlogsync/db"
)

func atoi32(s string) int32 {
	if i, err := strconv.Atoi(s); err != nil {
		return 0
	} else {
		return int32(i)
	}
}
func atoi64(s string) int64 {
	if i, err := strconv.ParseInt(s, 10, 64); err != nil {
		return 0
	} else {
		return int64(i)
	}
}

func isSameMonth(s1 int64, s2 int64) bool {
	t1 := time.Unix(s1, 0)
	t2 := time.Unix(s2, 0)
	if t1.Year()*100+int(t1.Month()) == t2.Year()*100+int(t2.Month()) {
		return true
	}
	return false
}

func main() {
	sync, err := newLogSync()
	if err != nil {
		log.Fatalln(err)
	}
	sync.run()
	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)
	select {
	case s := <-sg:
		log.Println("[main] got signal", s)
		sync.shutDown()
	}
	log.Printf("[main] quit")
}

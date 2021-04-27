package main

import (
	"bufio"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"
)

func newClient(id int, token string, addr string, scheme string) {

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: scheme, Host: addr, Path: "/v3/stream"}
	log.Printf("%d connecting to %s", id, u.String())

	header := http.Header{}
	header.Add("TOKEN", token)
	c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		log.Printf("%d dial failed:%s\n", id, err)
		return
	} else {
		log.Printf("%d connected to %s", id, u.String())
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("%d recv: %s", id, message)
		}
	}()

	//ticker := time.NewTicker(time.Second)
	//defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		//case t := <-ticker.C:
		//	err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
		//	if err != nil {
		//		log.Println("write:", err)
		//		return
		//	}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("%d write close: %s\n", id, err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Please input scheme, host & token file")
		return
	}
	scheme := os.Args[1]
	host := os.Args[2]
	tokenFile := os.Args[3]

	file, err := os.Open(tokenFile)
	check(err)
	defer file.Close()

	var tokens []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		tokens = append(tokens, scanner.Text())
	}

	if err = scanner.Err(); err != nil {
		panic(err)
	}

	for i, t := range tokens {
		token := t
		i2 := i
		go func() {
			newClient(i2, token, host, scheme)
		}()
		time.Sleep(10 * time.Millisecond)
	}
	select {}
}

package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var bufPool = sync.Pool{
	New: func() interface{} { return make([]byte, 32768) },
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  32768,
	WriteBufferSize: 32768,
}

var indexHTML []byte

func init() {
	var err error
	indexHTML, err = os.ReadFile("/app/index.html")
	if err != nil {
		indexHTML = []byte("Cloud Node Active")
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "7860"
	}

	targetURL, _ := url.Parse("http://127.0.0.1:5555")
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/moz") {
			if r.Header.Get("Sec-WebSocket-Key") != "" {
				handleWS(w, r)
				return
			}
			proxy.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/sub" {
			host := r.Host
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "trojan://1q2w3e4r5t@%s:443?type=ws&security=tls&path=%%2Fmoz#NyxServer-Trojan-WS\n", host)
			fmt.Fprintf(w, "trojan://1q2w3e4r5t@%s:443?type=httpupgrade&security=tls&path=%%2Fmoz#NyxServer-Trojan-HTTPUpgrade\n", host)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(indexHTML)
	})

	log.Printf("Gateway starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer clientConn.Close()

	sbConn, err := net.Dial("tcp", "127.0.0.1:5555")
	if err != nil {
		return
	}
	defer sbConn.Close()

	handshake := "GET /moz HTTP/1.1\r\n" +
		"Host: 127.0.0.1:5555\r\n" +
		"Connection: Upgrade\r\n" +
		"Upgrade: websocket\r\n\r\n"
	if _, err = sbConn.Write([]byte(handshake)); err != nil {
		return
	}

	reader := bufio.NewReader(sbConn)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			return
		}
		if len(line) == 0 {
			break
		}
	}

	errChan := make(chan error, 2)

	go func() {
		for {
			messageType, r, err := clientConn.NextReader()
			if err != nil {
				errChan <- err
				return
			}
			if messageType == websocket.BinaryMessage || messageType == websocket.TextMessage {
				buf := bufPool.Get().([]byte)
				_, err = io.CopyBuffer(sbConn, r, buf)
				bufPool.Put(buf)
				if err != nil {
					errChan <- err
					return
				}
			}
		}
	}()

	go func() {
		buf := bufPool.Get().([]byte)
		defer bufPool.Put(buf)
		for {
			n, err := reader.Read(buf)
			if err != nil {
				errChan <- err
				return
			}
			if n > 0 {
				if err = clientConn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					errChan <- err
					return
				}
			}
		}
	}()

	<-errChan
}

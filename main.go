package main

import (
	"bytes"
	"flag"
	"image/jpeg"
	"log"
	"net/url"
	"os"
	"time"
	"net/http"
	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
)

var display int
var addr string

func init() {
	flag.StringVar(&addr, "addr", "13.232.17.3:8080", "server address")
	flag.IntVar(&display, "display", 0, "number of the display to stream")
}

func main() {
	//hitting the api for setting path
	//taking path as parameter
	pathAsArg := os.Args[1]
	//log.Printf(pathAsArg)
	_, err := http.Get("http://13.232.17.3:8080/path/?servingPath="+pathAsArg)
    if err != nil {
        log.Printf("The HTTP request failed with error %s\n", err)
    } else {

		flag.Parse()
		interrupt := make(chan os.Signal, 1)
		//signal.Notify(interrupt, os.Interrupt)

		u := url.URL{Scheme: "ws", Host: addr, Path: "/source"}
		log.Printf("connecting to %s", u.String())

		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Fatal("dial:", err)
		}
		defer c.Close()

		n := screenshot.NumActiveDisplays()
		if display >= n {
			log.Fatalf("Display devices greater than that available. Enter value between [0-%d]", n)
		}

		sendPngBytes(c, display, interrupt)
	}
}

func sendPngBytes(conn *websocket.Conn, displayCount int, interrupt chan os.Signal) {
	bounds := screenshot.GetDisplayBounds(display)
	buf := new(bytes.Buffer)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:
			img, err := screenshot.CaptureRect(bounds)
			if err != nil {
				panic(err)
			}
			jpeg.Encode(buf, img, &jpeg.Options{15})
			//png.Encode(buf,img)
			bytesToSend := buf.Bytes()
			log.Printf("Start sending  stream. Size: %d Kb\n", len(bytesToSend)/1000)
			err = conn.WriteMessage(websocket.BinaryMessage, bytesToSend)
			log.Println("Done sending  stream.")
			if err != nil {
				log.Println("write:", err)
				return
			}
			buf.Reset()
		case <-interrupt:
			log.Println("Terminating stream..")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-time.After(time.Second):
			}
			return
		}
	}
}

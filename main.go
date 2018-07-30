package main

import (
	"bytes"
	"flag"
	"image/jpeg"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
	"strconv"
)

var display int
var addr string
var quality int
var fps int

func init() {
	flag.StringVar(&addr, "addr", "vps.seemyscreen.com.au:8081", "server address")
	flag.IntVar(&display, "display", 0, "number of the display to stream")
}

func main() {
	//hitting the api for setting path
	//taking path as parameter
	pathAsArg := os.Args[1]
	quality,_ = strconv.Atoi(os.Args[2])
	fps,_ = strconv.Atoi(os.Args[3])
	//log.Printf(pathAsArg)
	/*_, err := http.Get("http://vps.seemyscreen.com.au:8080/path/?servingPath=" + pathAsArg)
	if err != nil {
		log.Printf("The HTTP request failed with error %s\n", err)
	} else {*/

		flag.Parse()
		interrupt := make(chan os.Signal, 1)
		//signal.Notify(interrupt, os.Interrupt)

		u := url.URL{Scheme: "ws", Host: addr, Path: "/source/"+pathAsArg}
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
	//}
}

func sendPngBytes(conn *websocket.Conn, displayCount int, interrupt chan os.Signal) {
	bounds := screenshot.GetDisplayBounds(display)
	buf := new(bytes.Buffer)
	var frames = int64(1000/fps)
	ticker := time.NewTicker(time.Duration(frames) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:
			img, err := screenshot.CaptureRect(bounds)
			if err != nil {
				panic(err)
			}
			jpeg.Encode(buf, img, &jpeg.Options{quality})
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

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var (
	boundry  = "spiderman"
	m        = image.NewRGBA(image.Rect(0, 0, 640, 480))
	writeMut = sync.Mutex{}

	numOnline int64 = 0
)

func init() {
	blue := color.RGBA{0, 0, 255, 255}
	draw.Draw(m, m.Bounds(), &image.Uniform{blue}, image.ZP, draw.Src)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<form method="get" action="/click"><input type="image" name="imgbtn" src="/image"></form>`)
}

func mjpegHandler(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&numOnline, 1)

	n, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "cannot stream", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Cache-Control", "private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-type", "multipart/x-mixed-replace; boundary="+boundry)

	for {
		fmt.Fprintf(w, "--%s\n", boundry)
		fmt.Fprint(w, "Content-type: image/jpeg\n\n")

		writeMut.Lock()
		m.Set(120, 120, color.RGBA{255, 255, 0, 255})
		jpeg.Encode(w, m, &jpeg.Options{Quality: 70})
		fmt.Fprint(w, "\n\n")
		writeMut.Unlock()

		t := time.NewTimer(300 * time.Millisecond)

		select {
		case <-n.CloseNotify():
			atomic.AddInt64(&numOnline, -1)
			fmt.Println("...closed")
			return
		case <-t.C:
			fmt.Print(numOnline, " ")
		}
	}
}

func clickHandler(w http.ResponseWriter, r *http.Request) {
	x, _ := strconv.Atoi(r.FormValue("imgbtn.x"))
	y, _ := strconv.Atoi(r.FormValue("imgbtn.y"))

	writeMut.Lock()
	for xx := -1; xx <= 1; xx++ {
		for yy := -1; yy <= 1; yy++ {
			m.Set(x+xx, y+yy, color.RGBA{255, 255, 0, 255})
		}
	}
	writeMut.Unlock()
	fmt.Println(x, y)
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/click", clickHandler)
	http.HandleFunc("/image", mjpegHandler)
	err := http.ListenAndServe(":4040", nil)
	if err != nil {
		panic(err)
	}
}

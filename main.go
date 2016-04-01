package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var (
	httpPort = flag.Uint("http-port", 8080, "http port to run on")
)

var (
	boundry = "spiderman"
	m       = image.NewRGBA(image.Rect(0, 0, 800, 800))
	mut     = sync.RWMutex{}

	numOnline int64

	chanMap = make(map[chan bool]bool)
)

func init() {
	flag.Parse()
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<form method="get" action="/click">
	<input type="image" name="imgbtn" src="/image">
	
	<br />
	
	<h5>Color</h5>
	<input type="color" name="color">
	
	<br />
	
	<h5>Size</h5>
	<input type="range" name="size" min="1" max="99" value="3">
</form>`)
}

func mjpegHandler(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&numOnline, 1)

	mut.Lock()
	update := make(chan bool)
	chanMap[update] = true
	mut.Unlock()

	n, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "cannot stream", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Cache-Control", "private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-type", "multipart/x-mixed-replace; boundary="+boundry)

	flush := 10

	for {
		fmt.Fprintf(w, "--%s\n", boundry)
		fmt.Fprint(w, "Content-type: image/jpeg\n\n")

		mut.RLock()
		m.Set(120, 120, color.RGBA{255, 255, 0, 255})
		jpeg.Encode(w, m, &jpeg.Options{Quality: 70})
		fmt.Fprint(w, "\n\n")
		mut.RUnlock()

		t := time.NewTimer(15 * time.Second)

		if flush == 0 {
			select {
			case <-n.CloseNotify():
				atomic.AddInt64(&numOnline, -1)
				fmt.Println("...closed")

				mut.Lock()
				delete(chanMap, update)
				mut.Unlock()

				return
			case <-update:
				flush = 2
				fmt.Print("u")
			case <-t.C:
				fmt.Print(atomic.LoadInt64(&numOnline), " ")
			}
		} else {
			flush--
		}
	}
}

func clickHandler(w http.ResponseWriter, r *http.Request) {
	x, err := strconv.Atoi(r.FormValue("imgbtn.x"))
	if err != nil {
		http.Error(w, "invalid int for x", 400)
		return
	}

	y, err := strconv.Atoi(r.FormValue("imgbtn.y"))
	if err != nil {
		http.Error(w, "invalid int for y", 400)
		return
	}

	c := color.RGBA{255, 255, 0, 255}

	s, err := strconv.Atoi(r.FormValue("size"))
	if err != nil {
		http.Error(w, "invalid int for size", 400)
		return
	}

	if s < 0 || s > 100 {
		http.Error(w, "invalid size", 400)
		return
	}

	fc := r.FormValue("color")
	if fc != "" {
		if len(fc) != 7 || fc[0] != '#' {
			http.Error(w, "invalid color", 400)
			return
		}

		cr, err1 := strconv.ParseUint(fc[1:3], 16, 8)
		cg, err2 := strconv.ParseUint(fc[3:5], 16, 8)
		cb, err3 := strconv.ParseUint(fc[5:7], 16, 8)

		if err1 != nil || err2 != nil || err3 != nil {
			http.Error(w, "invalid color", 400)
			return
		}

		c = color.RGBA{uint8(cr), uint8(cg), uint8(cb), 255}
	}

	mut.Lock()
	for xx := 0 - s; xx <= s; xx++ {
		for yy := 0 - s; yy <= s; yy++ {
			m.Set(x+xx, y+yy, c)
		}
	}
	mut.Unlock()

	fmt.Println(x, y)
	w.WriteHeader(http.StatusNoContent)

	for k := range chanMap {
		k <- true
	}
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/click", clickHandler)
	http.HandleFunc("/image", mjpegHandler)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), nil)
	if err != nil {
		panic(err)
	}
}

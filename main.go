package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	Sessions = make(map[string]*Session)
	Results = make(map[string]*Result)
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/", index).Methods("GET")
	r.HandleFunc("/{id}/{c}.png", selection).Methods("GET")
	r.HandleFunc("/{id}/{key}/{n}.jpg", image).Methods("GET")
	r.HandleFunc("/{id}/verify", verify).Methods("GET")
	r.HandleFunc("/{id}/result", result).Methods("GET")
	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))),
	)
	r.PathPrefix("/img/").Handler(
		http.StripPrefix("/img/", http.FileServer(http.Dir("./img/"))),
	)
	log.Println("Serving at :8080")
	log.Println(http.ListenAndServe(":8080", r))
}

func index(w http.ResponseWriter, r *http.Request) {
	log.Println("IN " + r.URL.Path)
	defer log.Println("OUT " + r.URL.Path)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}
	_, err := w.Write([]byte(pageHead))
	if err != nil {
		return
	}
	flusher.Flush()
	s := NewSession()
	AddSession(s)
	defer RemoveSession(s)
	// Send session-specific CSS
	w.Write([]byte("<style>\n"))
	w.Write([]byte("button:active{content:url(" + s.ID + "/verify)}\n"))
	s.WriteState(w)
	w.Write([]byte("</style>\n"))
	flusher.Flush()
	for {
		var err error
		select {
		case b := <-s.Chan:
			// Listen for incoming HTML
			_, err = w.Write(b)
		case _ = <-s.Close:
			// Listen for session close signal
			return
		case <-time.After(5 * time.Second):
			// This is just to keep the session alive or close it when the
			// client stops reading bytes.
			_, err = w.Write([]byte(" "))
		}
		if err != nil {
			return
		}
		flusher.Flush()
	}
}

// Browser performs these requests when images are clicked
func selection(w http.ResponseWriter, r *http.Request) {
	log.Println("GET " + r.URL.Path)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	vars := mux.Vars(r)
	// Grab image id and parse int
	c := vars["c"]
	x, err := strconv.ParseUint(c, 10, 16)
	if err != nil {
		return
	}
	// Grab session
	id := vars["id"]
	SessionsLock.RLock()
	s, ok := Sessions[id]
	SessionsLock.RUnlock()
	if !ok {
		return
	}
	// Select image
	s.SelectImage(x)
}

// Serve image file from /sessionid/imgkey/imgcount.jpg path
func image(w http.ResponseWriter, r *http.Request) {
	log.Println("GET " + r.URL.Path)
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	vars := mux.Vars(r)
	// Grab session
	id := vars["id"]
	SessionsLock.RLock()
	s, ok := Sessions[id]
	SessionsLock.RUnlock()
	if !ok {
		return
	}
	ks := vars["key"]
	k, err := strconv.ParseUint(ks, 10, 16)
	if err != nil {
		return
	}
	http.ServeFile(w, r, s.Images[k])
}

// Browser performs this request when Verify button is clicked
func verify(w http.ResponseWriter, r *http.Request) {
	log.Println("GET " + r.URL.Path)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	vars := mux.Vars(r)
	// Grab session
	id := vars["id"]
	SessionsLock.RLock()
	s, ok := Sessions[id]
	SessionsLock.RUnlock()
	if !ok {
		return
	}
	// Set result by ID
	ResultsLock.Lock()
	Results[s.ID] = s.GetResult()
	ResultsLock.Unlock()
	// Send HTML to refresh page to results
	s.Chan <- []byte(fmt.Sprintf(
		"<meta http-equiv=\"refresh\" content=\"0; url=%s/result\">\n",
		s.ID,
	))
	// Signal to close client connection
	s.Close <- struct{}{}
}

// Render captcha results
func result(w http.ResponseWriter, r *http.Request) {
	log.Println("GET " + r.URL.Path)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	vars := mux.Vars(r)
	// Grab session
	id := vars["id"]
	ResultsLock.RLock()
	x, ok := Results[id]
	ResultsLock.RUnlock()
	if !ok {
		return
	}
	ResultsLock.Lock()
	delete(Results, id)
	ResultsLock.Unlock()
	errors := 0
	w.Write([]byte("<fieldset><legend>Selection</legend>"))
	for _, x := range x.Selection {
		w.Write([]byte(fmt.Sprintf("<img width=\"100\" src=\"/%s\">", x)))
		if strings.Contains(x, "d") {
			errors++
		}
	}
	w.Write([]byte("</fieldset>"))
	w.Write([]byte("<fieldset><legend>Remaining</legend>"))
	for _, x := range x.Images {
		w.Write([]byte(fmt.Sprintf("<img width=\"100\" src=\"/%s\">", x)))
		if strings.Contains(x, "c") {
			errors++
		}
	}
	w.Write([]byte("</fieldset>"))
	w.Write([]byte(fmt.Sprintf("<div>Errors: %d", errors)))

}

// Main portion of captcha page HTML
const pageHead = `<html>
<head>
	<link rel="stylesheet" href="/static/style.css">
</head>
<body>
	<table>
		<tr>
			<th colspan="3">Select all images with cats.</th>
		</tr>
		<tr>
			<td class="c" id="c0"><div></div><div></div></td>
			<td class="c" id="c1"><div></div><div></div></td>
			<td class="c" id="c2"><div></div><div></div></td>
		</tr>
		<tr>
			<td class="c" id="c3"><div></div><div></div></td>
			<td class="c" id="c4"><div></div><div></div></td>
			<td class="c" id="c5"><div></div><div></div></td>
		</tr>
		<tr>
			<td class="c" id="c6"><div></div><div></div></td>
			<td class="c" id="c7"><div></div><div></div></td>
			<td class="c" id="c8"><div></div><div></div></td>
		</tr>
		<tr>
			<td colspan="3" style="padding: 0.5em; text-align:right">
				<button>Verify</button>
			</td>
		</tr>
	</table>
`

package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"sync"
)

// Session represents one instance of a captcha challenge/solution
type Session struct {
	ID        string
	Chan      chan []byte
	Close     chan struct{}
	State     uint16
	Images    [9]string
	Counts    [9]int
	Selection []string
}

// Sessions maps all sessions by ID
var Sessions map[string]*Session

// SessionsLock is mutex for Sessions
var SessionsLock sync.RWMutex

// AddSession adds *Session to mapping by ID
func AddSession(s *Session) {
	SessionsLock.Lock()
	Sessions[s.ID] = s
	SessionsLock.Unlock()
}

// RemoveSession removes existing *Session
func RemoveSession(s *Session) {
	SessionsLock.Lock()
	delete(Sessions, s.ID)
	SessionsLock.Unlock()
}

// NewSession returns a new *Session instance with random ID
func NewSession() *Session {
	b := make([]byte, 8)
	rand.Read(b)
	id := hex.EncodeToString(b)
	return &Session{
		ID:    id,
		Chan:  make(chan []byte),
		Close: make(chan struct{}),
		State: 0,
	}
}

// NewImage generates a new image and updates Images and Counts
func (s *Session) NewImage(i int) string {
	s.Images[i] = randImg()
	count := s.Counts[i]
	s.Counts[i]++
	return fmt.Sprintf("url(%s/%d/%d.jpg)", s.ID, i, count)
}

// GetResult returns the results of a Session
func (s *Session) GetResult() *Result {
	return &Result{
		Images:    s.Images,
		Selection: s.Selection,
	}
}

// WriteState writes the current captcha state CSS
func (s *Session) WriteState(w io.Writer) {
	for i := 0; i < 9; i++ {
		c := s.NewImage(i)
		w.Write([]byte(fmt.Sprintf(
			"#c%d>div:nth-child(1){background-image:%s}\n",
			i,
			c,
		)))
		w.Write([]byte(fmt.Sprintf(
			"#c%d>div:nth-child(1):active{content:url(%s/%d.png)}\n",
			i,
			s.ID,
			i,
		)))
	}
}

// SelectImage manages the behavior when a particular image is clicked
func (s *Session) SelectImage(x uint64) {
	sel := s.Images[x]
	s.Selection = append(s.Selection, sel)
	img := s.NewImage(int(x))
	var s0, s1 int
	if s.State&(1<<x) == 0 {
		s0 = 1
	} else {
		s0 = 2
	}
	s.State ^= (1 << x)
	if s.State&(1<<x) == 0 {
		s1 = 1
	} else {
		s1 = 2
	}
	str := fmt.Sprintf(`<style>
	#c%d>div:nth-child(%d){display:none}
	#c%d>div:nth-child(%d){display:block;background-image:%s}
	#c%d>div:nth-child(%d):active{content:url(%s/%d.png)}
	</style>`, x, s0, x, s1, img, x, s1, s.ID, x)
	s.Chan <- []byte(str)
}

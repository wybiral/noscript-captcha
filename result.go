package main

import "sync"

// Results maps Session ID to *Result
var Results map[string]*Result

// ResultsLock is mutex for Results
var ResultsLock sync.RWMutex

// Result represents one captcha result/submission
type Result struct {
	Images    [9]string
	Selection []string
}

package utils

import (
	"bytes"
	"io"
	"log"
	"strings"
	"sync"
	"time"
)

type TtyAutoInput struct {
	mu           sync.Mutex
	match        string
	value        string
	outputOldLen int
	output       bytes.Buffer
	reading      bytes.Buffer
	waitCh       chan struct{}
}

func NewTtyAutoInput(match, value string) *TtyAutoInput {
	return &TtyAutoInput{
		match:  match,
		value:  value,
		waitCh: make(chan struct{}, 1),
	}
}

func (autoS *TtyAutoInput) Read(p []byte) (n int, err error) {
	time.Sleep(time.Second)
	var output string
	autoS.mu.Lock()
	b := autoS.output.Bytes()
	if autoS.outputOldLen == len(b) {

		// check output
		if autoS.reading.Len() == 0 {
			for {
				autoS.mu.Unlock()
				_, ok := <-autoS.waitCh
				if !ok {
					return 0, io.EOF
				}
				autoS.mu.Lock()
				if autoS.outputOldLen != autoS.output.Len() || autoS.reading.Len() > 0 {
					b = autoS.output.Bytes()
					autoS.outputOldLen = len(b)
					break
				}
			}
		} else {
			autoS.mu.Unlock()
			// write the exiting reading buffer to stdin
			n, err = autoS.reading.Read(p)
			return
		}

	} else {
		autoS.outputOldLen = len(b)
	}
	autoS.mu.Unlock()

	const lenMax = 200
	if len(b) > lenMax {
		output = string(b[len(b)-lenMax:])
	} else {
		output = string(b)
	}
	output = strings.TrimRight(output, "\n")
	//idx := strings.LastIndex(output, "\n")
	//if idx > 0 {
	//	output = output[idx:]
	//}
	log.Println("output=", output)
	if autoS.reading.Len() == 0 {
		if strings.Contains(output, autoS.match) {
			autoS.reading.WriteString(autoS.value + "\n")
			log.Println("write", autoS.value)
		}
	}
	n, err = autoS.reading.Read(p)
	return
}

func (autoS *TtyAutoInput) Write(p []byte) (n int, err error) {
	autoS.mu.Lock()
	n, err = autoS.output.Write(p)
	autoS.mu.Unlock()
	if err != nil {
		close(autoS.waitCh)
	} else {
		select {
		case autoS.waitCh <- struct{}{}:
		default:
		}
	}
	return
}

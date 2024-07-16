package main

import (
	"bufio"
	"container/ring"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func logger(wg *sync.WaitGroup, log <-chan string) {

	var logStream *os.File = os.Stdout

	for {
		in := <-log
		switch in {
		case "end of log":
			fmt.Fprintln(logStream, in)
			wg.Done()
			return

		default:
			fmt.Fprintln(logStream, in)
		}
	}
}

func Positive(log chan<- string, wg *sync.WaitGroup, done <-chan int, cIn <-chan int) <-chan int {
	cOut := make(chan int)
	wg.Add(1)
	go func() {
		for {
			select {
			case <-done:
				log <- "stop Positive"
				wg.Done()
				return

			case el := <-cIn:
				if el >= 0 {
					log <- fmt.Sprintf("Positive stage passed by %d", el)
					cOut <- el
				} else {
					log <- fmt.Sprintf("Positive stage failed by %d", el)
				}
			}
		}
	}()

	return cOut
}

func Trine(log chan<- string, wg *sync.WaitGroup, done <-chan int, cIn <-chan int) <-chan int {
	cOut := make(chan int)
	wg.Add(1)
	go func() {
		for {
			select {
			case <-done:
				log <- "stop Trine"
				wg.Done()
				return

			case el := <-cIn:
				if el%3 == 0 && el != 0 {
					log <- fmt.Sprintf("Trine stage passed by %d", el)
					cOut <- el
				} else {
					log <- fmt.Sprintf("Trine stage failed by %d", el)
				}
			}
		}
	}()

	return cOut
}

var ringDelay int = 2

var ringSize int = 5

func RingBuf(log chan<- string, wg *sync.WaitGroup, done <-chan int, cIn <-chan int) <-chan int {
	r := newrBuf(ringSize)
	cOut := make(chan int)
	wg.Add(1)
	go func() {

		for {
			select {
			case <-done:
				log <- "stop RingBuf"
				wg.Done()
				return

			case el := <-cIn:
				r.Pull(el)
				log <- fmt.Sprintf("%d added to RingBuf", el)

			case <-time.After(time.Second * time.Duration(ringDelay)):
				for _, i := range r.Get() {
					log <- fmt.Sprintf("RingBuf release: %d", i)
					cOut <- i
				}
			}
		}
	}()
	return cOut
}

type rBuf struct {
	dataRing *ring.Ring
}

func newrBuf(size int) *rBuf {

	return &rBuf{
		ring.New(size),
	}
}

func (r *rBuf) Pull(el int) {
	r.dataRing.Value = el
	r.dataRing = r.dataRing.Next()
}

func (r *rBuf) Get() []int {
	resultArr := make([]int, 0)
	size := r.dataRing.Len()

	for i := 0; i < size; i++ {
		if r.dataRing.Value == nil {
			r.dataRing = r.dataRing.Next()
			continue
		}
		resultArr = append(resultArr, r.dataRing.Value.(int))
		r.dataRing.Value = nil
		r.dataRing = r.dataRing.Next()

	}
	return resultArr
}

var ErrNoInts = fmt.Errorf("there is no ints to process")

var ErrWrongCmd = fmt.Errorf("unsupported command, you can use commands: input, quit")

type DataSrc struct {
	data *[]int
	C    chan int
}

func NewDataSrc() *DataSrc {
	d := make([]int, 0)
	return &DataSrc{&d,
		make(chan int),
	}
}

func (d DataSrc) intInput(i int) {
	*d.data = append(*d.data, i)
}

func (d DataSrc) process() error {

	if len(*d.data) == 0 {
		return ErrNoInts
	}
	fmt.Println("Processing...")
	for _, i := range *d.data {
		d.C <- i
	}

	*d.data = make([]int, 0)
	return nil
}

func (d DataSrc) ScanCmd(log chan<- string, done chan int) {

	for {
		var cmd string
		fmt.Println("Enter command:")
		fmt.Scanln(&cmd)
		log <- fmt.Sprintf("command received: %s", cmd)

		switch cmd {

		case "input":
			log <- "INPUT NUMBERS mode"
			fmt.Printf("Enter numbers separated by whitespaces:\n")
			in := bufio.NewScanner(os.Stdin)
			in.Scan()
			anything := in.Text()
			log <- fmt.Sprintf("string received: %s", anything)

			someSlice := strings.Fields(anything)

			for _, some := range someSlice {

				if num, err := strconv.ParseInt(some, 10, 0); err == nil {

					log <- fmt.Sprintf("number sent to processing: %d", (int(num)))
					d.intInput(int(num))
				}
			}

			err := d.process()
			if err != nil {
				fmt.Println(err)
				log <- fmt.Sprintf("error: %s", err)
				break
			}
			time.Sleep(time.Second * time.Duration(ringDelay+1))

		case "quit":
			log <- "sending DONE signal and stop ScanCmd"
			close(done)
			return

		default:
			log <- fmt.Sprintf("error: %s", ErrWrongCmd)
			fmt.Println(ErrWrongCmd)
		}
	}
}

func receiver(log chan<- string, wg *sync.WaitGroup, done chan int, c <-chan int) {
	for {
		select {
		case <-done:
			log <- "stop receiver"
			wg.Done()
			return

		case el := <-c:
			log <- fmt.Sprintf("number received by receiver: %d", el)
			fmt.Printf("Int received: %d\n", el)
		}
	}
}

func main() {
	var wg sync.WaitGroup

	logChan := make(chan string)
	go logger(&wg, logChan)
	logChan <- "program started"

	d := NewDataSrc()
	done := make(chan int)
	pipeline := RingBuf(logChan, &wg, done, Trine(logChan, &wg, done, Positive(logChan, &wg, done, d.C)))

	go d.ScanCmd(logChan, done)
	wg.Add(1)
	go receiver(logChan, &wg, done, pipeline)
	wg.Wait()

	wg.Add(1)
	logChan <- "end of log"
	wg.Wait()
}

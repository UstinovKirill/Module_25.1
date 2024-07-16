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

func Positive(wg *sync.WaitGroup, done <-chan int, cIn <-chan int) <-chan int {
	cOut := make(chan int)
	wg.Add(1)
	go func() {
		for {
			select {
			case <-done:
				wg.Done()
				return

			case el := <-cIn:
				if el >= 0 {
					cOut <- el
				}
			}
		}
	}()

	return cOut
}

func Trine(wg *sync.WaitGroup, done <-chan int, cIn <-chan int) <-chan int {
	cOut := make(chan int)
	wg.Add(1)
	go func() {
		for {
			select {
			case <-done:
				wg.Done()
				return

			case el := <-cIn:
				if el%3 == 0 && el != 0 {
					cOut <- el
				}
			}
		}
	}()

	return cOut
}

var ringDelay int = 2

var ringSize int = 5

func RingBuf(wg *sync.WaitGroup, done <-chan int, cIn <-chan int) <-chan int {
	r := newrBuf(ringSize)
	cOut := make(chan int)
	wg.Add(1)
	go func() {

		for {
			select {
			case <-done:
				wg.Done()
				return

			case el := <-cIn:
				r.Pull(el)

			case <-time.After(time.Second * time.Duration(ringDelay)):
				for _, i := range r.Get() {
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

var ErrNoInts = fmt.Errorf("There is no ints to process")

var ErrWrongCmd = fmt.Errorf("Unsupported command. You can use commands: input, quit.")

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

func (d DataSrc) Process() error {

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

func (d DataSrc) ScanCmd(done chan int) {

	for {
		var cmd string
		fmt.Println("Enter command:")
		fmt.Scanln(&cmd)

		switch cmd {

		case "input":

			fmt.Printf("Enter numbers separated by whitespaces:\n")
			in := bufio.NewScanner(os.Stdin)
			in.Scan()
			anything := in.Text()

			someSlice := strings.Fields(anything)

			for _, some := range someSlice {

				if num, err := strconv.ParseInt(some, 10, 0); err == nil {

					d.intInput(int(num))
				}
			}

			err := d.Process()
			if err != nil {
				fmt.Println(err)
				break
			}
			time.Sleep(time.Second * time.Duration(ringDelay+1))

		case "quit":
			close(done)

			return

		default:
			fmt.Println(ErrWrongCmd)
		}
	}
}

func receiver(wg *sync.WaitGroup, done chan int, c <-chan int) {
	for {
		select {
		case <-done:
			wg.Done()
			return

		case el := <-c:
			fmt.Printf("Int received: %d\n", el)
		}

	}
}

func main() {
	var wg sync.WaitGroup

	d := NewDataSrc()
	done := make(chan int)
	pipeline := RingBuf(&wg, done, Trine(&wg, done, Positive(&wg, done, d.C)))

	go d.ScanCmd(done)
	wg.Add(1)
	go receiver(&wg, done, pipeline)
	wg.Wait()
}

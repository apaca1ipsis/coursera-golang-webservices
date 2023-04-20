package main

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	prevChan := make(chan interface{})
	defer close(prevChan)
	for _, worker := range jobs {
		wg.Add(1)
		newChan := make(chan interface{})
		go func(in, out chan interface{}, worker job, wg *sync.WaitGroup) {
			defer wg.Done()
			worker(in, out)
			close(out)
		}(prevChan, newChan, worker, wg)
		prevChan = newChan
	}
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	quotaCh := make(chan struct{}, 1)
	for dataRaw := range in {
		data := strconv.Itoa(dataRaw.(int))
		//fmt.Println("SingleHash. got - ", data)

		answ := make(chan string, 2)

		go crc32Calc(answ, data)
		go md5Calc(data, quotaCh, answ)
		wg.Add(1)
		go combine(answ, out, wg)
	}
	wg.Wait()
}

func combine(answerCh chan string, outerCh chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	first := <-answerCh
	second := <-answerCh
	fmt.Println("combine. ready - ", first, "~", second)
	outerCh <- first + "~" + second
}
func crc32Calc(ch chan string, val string) {
	res := DataSignerCrc32(val)
	ch <- res

}

func md5Calc(s string, quotaCh chan struct{}, ch chan string) {
	quotaCh <- struct{}{}
	resCh := make(chan string)
	go func(s string, out chan string) {
		v := DataSignerMd5(s)
		out <- v
	}(s, resCh)
	res := <-resCh
	<-quotaCh
	go crc32Calc(ch, res)
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	runtime.GOMAXPROCS(1)
	for dataRaw := range in {
		data, ok := dataRaw.(string)
		if !ok {
			panic("cannot convert input to string")
		}
		wg.Add(1)
		go thData(data, out, wg)
		runtime.Gosched()
	}
	wg.Wait()
}

func thData(s string, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()

	wgInner := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	thS := make([]string, 6)
	for i := 0; i <= 5; i++ {
		wgInner.Add(1)

		go func(i int, s string, wgInner *sync.WaitGroup, arr []string, mu *sync.Mutex) {
			defer wgInner.Done()
			res := DataSignerCrc32(strconv.Itoa(i) + s)
			mu.Lock()
			arr[i] = res
			mu.Unlock()
		}(i, s, wgInner, thS, mu)
	}
	wgInner.Wait()

	res := strings.Join(thS, "")
	fmt.Printf("result for %v is %v\n", s, res)

	out <- res
}

func CombineResults(in, out chan interface{}) {
	var res []string
	for dataRaw := range in {
		data, ok := dataRaw.(string)
		if !ok {
			panic("cannot convert input to string")
		}
		res = append(res, data)
	}
	sort.Strings(res)

	out <- strings.Join(res, "_")
}

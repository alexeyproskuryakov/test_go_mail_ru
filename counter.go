package main

import (
	"bufio"
	"os"
	"fmt"
	"strings"
	"regexp"
	"net/http"
	"io/ioutil"
	"sync"
	"flag"
)


const URL_REGEX = `https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,4}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)`

type void struct{}

type urlProcessObject struct {
	Url  string
	Wait *sync.WaitGroup
}

type processContext struct {
	StackChan chan void
	Counter   *counter
	Q         string
}

type counter struct {
	count int
	lock  *sync.Mutex
}


func newCounter() *counter {
	return &counter{lock:&sync.Mutex{}}
}

func (c *counter) Add(count int) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.count += count
}

func (c *counter) GetCount() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.count
}


func RetrieveInputUrls() chan urlProcessObject {
	scanner := bufio.NewScanner(os.Stdin)
	urlsToProcessChan := make(chan urlProcessObject)
	go func() {
		var waitGroup sync.WaitGroup
		for {
			url := strings.TrimSpace(scanner.Text())
			if matched, err := regexp.MatchString(URL_REGEX, url); err == nil && matched {
				waitGroup.Add(1)
				urlsToProcessChan <- urlProcessObject{Url:url, Wait:&waitGroup}
			}

			if !scanner.Scan() {
				break
			}
		}
		waitGroup.Wait()
		close(urlsToProcessChan)

	}()
	return urlsToProcessChan
}

func loadBody(url string) (*string, error) {
	res, err := http.Get(url)
	if err != nil {
		fmt.Errorf("ERROR: Can not load url %s\n", url)
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		fmt.Errorf("ERROR: Can not read data from url: %s\n", url)
		return nil, err
	}
	result := string(body)
	return &result, nil
}

func processBody(data, q string) int {
	return strings.Count(strings.ToLower(data), q)
}


func processUrl(url urlProcessObject, ctx processContext) {
	data, err := loadBody(url.Url)
	if err != nil {
		return
	}
	count := processBody(*data, ctx.Q)
	ctx.Counter.Add(count)

	url.Wait.Done()
	<-ctx.StackChan

	fmt.Printf("Count for %s: %d\n", url.Url, count)
}

func ProcessUrls(urls chan urlProcessObject, k int, q string) {
	counter := newCounter()
	stackChan := make(chan void, k)

	for url := range urls {
		stackChan <- void{} //if stack is full will wait for not start another goroutines
		go processUrl(url, processContext{StackChan:stackChan, Counter:counter, Q:q})
	}

	close(stackChan)

	fmt.Printf("Total: %d\n", counter.GetCount())
}

func main() {
	k := flag.Int("k", 5, "max count of threads")
	q := flag.String("q", "go", "query string in urls body")
	flag.Parse()
	ProcessUrls(RetrieveInputUrls(), *k, *q)
}

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"strings"
	"flag"
	"sync"
)

// Multiplexer Proxy
type MuxProxy struct {
	urls    []*url.URL
	proxies []*httputil.ReverseProxy
	index   int
	max     int
}

// Create a new Multiplexer Proxy, and initializes the data with an array of URLs
func NewMuxProxy(rawurls []string) *MuxProxy {
	l := len(rawurls)
	urls := make([]*url.URL, l)
	proxies := make([]*httputil.ReverseProxy, l)
	for i, rawurl := range rawurls {
		u, e := url.Parse(rawurl)
		if e != nil {
			log.Fatal("URL Parse Error", e)
		}
		urls[i] = u
		proxies[i] = httputil.NewSingleHostReverseProxy(u)
	}
	return &MuxProxy{urls, proxies, 0, l}
}

// Proxy to server
func (p *MuxProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	index := p.Switcher()
	log.Println("To:", p.urls[index], "From:", r.RemoteAddr, "-", r.URL)
	p.proxies[index].ServeHTTP(w, r)
}

// Switches between proxies
func (p *MuxProxy) Switcher() int {
	p.index += 1
	if p.index == p.max {
		p.index = 0
	}
	return p.index
}

// Settings for the Multiplexer Proxy
type Settings struct {
	Host     string   `json:"host"`
	Protocol string   `json:"protocol"`
	Format   string   `json:"format"`
	Ports    []string `json:"ports"`
}

// Runs a command and prints out it's output
func RunCommand(cmd string, wg *sync.WaitGroup) {
	defer wg.Done()
	args := strings.Fields(cmd)
	id := args[2:3]
	app := exec.Command(args[0], args[1:]...)
	stdout, err := app.StdoutPipe()
	stderr, err := app.StderrPipe()
	if err != nil {
		log.Fatal(id, err)
	}
	if err := app.Start(); err != nil {
		log.Fatal(id, err)
	}
	// Prints out the output and errors for the app
	go ReadWrite(id, stdout)
	go ReadWrite(id, stderr)
	// Waits for the app to close
	app.Wait()
	fmt.Println(id, "Restarting Server")
	wg.Add(1)
	//go RunCommand(cmd, wg)
}

// Reads from the io.Reader and outputs the data with the id as the Header
func ReadWrite(id []string, out io.Reader) {
	rdr := bufio.NewReader(out)
	line := ""
	for {
		// Reads a line from the output
		buf, part, err := rdr.ReadLine()
		if err != nil {
			// Error Checking
			if err == io.EOF {
				continue
			}
			break
		}
		// Adds the line to temp variable
		line += fmt.Sprintf("%s", buf)
		// If the line isn't partial, print the temp var
		// else keep adding to the temp var
		if !part {
			fmt.Printf("%s> %s\n", id, line)
			line = ""
		}
	}
}

func MuxProxyParseJSON() (*MuxProxy, sync.WaitGroup) {
	// Parse files
	filedata, err := ioutil.ReadFile("settings.json")
	if err != nil {
		panic(err)
	}
	// JSON to Struct
	var options Settings
	if err := json.Unmarshal(filedata, &options); err != nil {
		panic(err)
	}
	// Dynamic programming stuff
	var wg sync.WaitGroup
	num_ports := len(options.Ports)
	cmds := make([]string, num_ports)
	urls := make([]string, num_ports)
	for i := 0; i < num_ports; i++ {
		urls[i] = fmt.Sprintf("%s://%s:%s",
			options.Protocol,
			options.Host,
			options.Ports[i])
		cmds[i] = fmt.Sprintf(options.Format,
			options.Ports[i])
		wg.Add(1)
		go RunCommand(cmds[i], &wg)
	}
	return NewMuxProxy(urls), wg
}

func main() {
	port := flag.String("port", "8080", "Port that the Reverse Proxy will run on")
	flag.Parse()
	// Server stuff
	mprox, wg := MuxProxyParseJSON()
	http.Handle("/", mprox)
	log.Printf("Load balancing on %s to %d different processes", *port, mprox.max)
	go http.ListenAndServe(":"+*port, nil)
	wg.Wait()
}
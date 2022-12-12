package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

// EC2 Meta data end point
const (
	baseUrl = "http://169.254.169.254/latest/meta-data"
)

var (
	port   = ""
	region = ""
	az     = ""
)

func main() {
	// Parse command line arguments
	flag.StringVar(&port, "port", getEnv("PORT_NUMBER", "8080"), "port number to listen")
	flag.Parse()

	// Fetch EC2 meta data
	if err := extractEc2Meta(); err == nil {
		log.Printf("Region: %v", region)
		log.Printf("AZ: %v", az)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", whoamiHandler)
	mux.HandleFunc("/api", apiHandler)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	log.Printf("Server starting on %s", server.Addr)
	log.Fatal(server.ListenAndServe())

}

// Extract EC2 Meta data
func extractEc2Meta() error {
	var err error
	// Region
	urlRegion := fmt.Sprintf("%v/placement/region", baseUrl)
	region, err = getMeta(urlRegion)

	// Availability Zone
	urlAz := fmt.Sprintf("%v/placement/availability-zone", baseUrl)
	az, err = getMeta(urlAz)

	return err
}

func whoamiHandler(w http.ResponseWriter, req *http.Request) {
	u, _ := url.Parse(req.URL.String())
	wait := u.Query().Get("wait")
	if len(wait) > 0 {
		duration, err := time.ParseDuration(wait)
		if err == nil {
			time.Sleep(duration)
		}
	}

	hostname, _ := os.Hostname()
	_, _ = fmt.Fprintln(w, "Hostname:", hostname)

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			_, _ = fmt.Fprintln(w, "IP:", ip)
		}
	}

	_, _ = fmt.Fprintln(w, "RemoteAddr:", req.RemoteAddr)
	if err := req.Write(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func apiHandler(w http.ResponseWriter, req *http.Request) {
	hostname, _ := os.Hostname()

	data := struct {
		Hostname string      `json:"hostname,omitempty"`
		Region   string      `json:"region,omitempty"`
		AZ       string      `json:"az,omitempty"`
		IP       []string    `json:"ip,omitempty"`
		Headers  http.Header `json:"headers,omitempty"`
		URL      string      `json:"url,omitempty"`
		Host     string      `json:"host,omitempty"`
		Method   string      `json:"method,omitempty"`
		Time     time.Time   `json:"time,omitempty"`
	}{
		Hostname: hostname,
		Region:   region,
		AZ:       az,
		IP:       []string{},
		Headers:  req.Header,
		URL:      req.URL.RequestURI(),
		Host:     req.Host,
		Method:   req.Method,
		Time:     time.Now(),
	}

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil {
				data.IP = append(data.IP, ip.String())
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getMeta(url string) (string, error) {

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

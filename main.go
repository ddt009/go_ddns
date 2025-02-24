package main

import (
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"golang.org/x/crypto/bcrypt"
	"log"
	"my-go-dns/aliddns"
	"net"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// Record structure definition
type Record struct {
	Host string `json:"host"`
	IPv6 string `json:"ipv6"`
	IPv4 string `json:"ipv4"`
	At   int64  `json:"at"`
}

// AliyunHost structure for Aliyun account information
type AliyunHost struct {
	AccessKeyID     string   `toml:"AccessKeyID"`
	AccessKeySecret string   `toml:"AccessKeySecret"`
	Region          string   `toml:"Region"`
	Domains         []string `toml:"Domains"`
}

// Config structure for configuration file
type Config struct {
	Password string                `toml:"password" json:"password,omitempty"`
	Aliyun   map[string]AliyunHost `toml:"aliyun" json:"aliyun,omitempty"`
}

var (
	configFile = "config.toml"
	config     Config
	cache      = make(map[string]Record)
	mu         sync.Mutex
)

func varDump(v interface{}) {
	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)
	fmt.Printf("Type: %s\n", typ)
	fmt.Printf("Value: %v\n", val)
}

// Load configuration
func loadConfig() {
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Printf("Configuration file does not exist, creating a new one: %s\n", configFile)
		config = Config{Aliyun: make(map[string]AliyunHost)}
		saveConfig()
		return
	}

	err = toml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Failed to parse configuration file: %v", err)
	}
}

// Save configuration
func saveConfig() {
	data, err := toml.Marshal(config)
	if err != nil {
		log.Fatalf("Failed to write configuration file: %v", err)
	}
	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		log.Fatalf("Failed to save configuration file: %v", err)
	}
}

// Get client IP
func getClientIP(r *http.Request) string {
	// 1. 检查 X-Forwarded-For 头部
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// X-Forwarded-For 可能包含多个 IP，用逗号分隔，取第一个有效 IP
		ips := strings.Split(forwarded, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip := net.ParseIP(ip); ip != nil {
				return ip.String()
			}
		}
	}

	// 2. 检查 X-Real-IP 头部
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		if ip := net.ParseIP(realIP); ip != nil {
			return ip.String()
		}
	}
	// 3. 使用 RemoteAddr
	remoteAddr := r.RemoteAddr
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		if parsedIP := net.ParseIP(host); parsedIP != nil {
			return parsedIP.String()
		}
	}
	// 如果无法解析任何有效 IP，返回空字符串
	return ""
}

// Hash password
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// Check password
func checkPassword(password string) bool {
	mu.Lock()
	defer mu.Unlock()

	if config.Password == "" {
		// Initialize password
		hash, err := hashPassword(password)
		if err != nil {
			log.Println("Failed to hash password:", err)
			return false
		}
		config.Password = hash
		saveConfig()
		log.Println("Password set and saved to configuration file")
		return true
	}

	// Verify password
	err := bcrypt.CompareHashAndPassword([]byte(config.Password), []byte(password))
	return err == nil
}

// Clean up expired records
func cleanupExpiredRecords() {
	now := time.Now().Unix()
	for host, record := range cache {
		if now-record.At > 86400 { // 24-hour validity
			delete(cache, host)
		}
	}
}

// Handle GET /
func handleRoot(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, getClientIP(r))
}

// Handle GET /list
func handleListGet(w http.ResponseWriter, r *http.Request) {
	html := `<html><body>
    <form action="/list" method="post">
        Password: <input type="password" name="password">
        <input type="submit" value="Submit">
    </form>
    </body></html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

// Handle POST /list
func handleListPost(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	password := r.FormValue("password")

	if !checkPassword(password) {
		//_, _ = fmt.Fprintf(w, "Your IP: %s\n", getClientIP(r))
		_, _ = fmt.Fprintf(w, getClientIP(r))
		return
	}

	// Get cached data and sort
	mu.Lock()
	defer mu.Unlock()

	var records []Record
	for _, v := range cache {
		if time.Now().Unix()-v.At <= 86400 { // 24-hour validity
			records = append(records, v)
		}
	}

	// Sort by timestamp in descending order
	sort.Slice(records, func(i, j int) bool {
		return records[i].At > records[j].At
	})

	// Format output as HTML table
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := "<html><body><table border='1'><tr><th>Host</th><th>IPv6</th><th>IPv4</th><th>Timestamp (UTC+8)</th></tr>"
	for _, record := range records {
		timestamp := time.Unix(record.At, 0).In(time.FixedZone("UTC+8", 8*3600)).Format("2006-01-02 15:04:05")
		html += fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>", record.Host, record.IPv6, record.IPv4, timestamp)
	}
	html += "</table></body></html>"
	_, _ = w.Write([]byte(html))
}

// Handle POST /
func handlePost(w http.ResponseWriter, r *http.Request) {
	var host, ipv6, ipv4 string

	if r.Header.Get("Content-Type") == "application/json" {
		// Parse JSON data
		var data map[string]string
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, "Invalid JSON data", http.StatusBadRequest)
			return
		}
		host = data["host"]
		ipv6 = data["ipv6"]
		ipv4 = getClientIP(r)
	} else {
		// Parse form data
		_ = r.ParseForm()
		host = r.FormValue("host")
		ipv6 = r.FormValue("ipv6")
		ipv4 = getClientIP(r)
	}

	if len(host) > 16 || len(ipv6) > 39 {
		//http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}

	// Clean up expired records
	mu.Lock()
	cleanupExpiredRecords()
	mu.Unlock()

	mu.Lock()
	defer mu.Unlock()

	// Check if update is needed
	record, exists := cache[host]
	if exists && record.IPv4 == ipv4 && record.IPv6 == ipv6 {
		//_, _ = fmt.Fprintf(w, "No changes detected")
		return
	}

	// Update cache
	cache[host] = Record{Host: host, IPv6: ipv6, IPv4: ipv4, At: time.Now().Unix()}

	// Check if Aliyun DNS update is needed
	aliyunHost, found := config.Aliyun[host]
	if found {
		for _, domain := range aliyunHost.Domains {
			go updateAliyunDNS(aliyunHost, domain, ipv6)
		}
	}

	//_, _ = fmt.Fprintf(w, "Submission successful")
}

// Update Aliyun DNS
func updateAliyunDNS(aliyunHost AliyunHost, domain, ipv6 string) {
	fmt.Printf("Calling Aliyun API to update IPv6 for %s to %s\n", domain, ipv6)

	manager, err := aliddns.NewDNSManager(aliyunHost.Region, aliyunHost.AccessKeyID, aliyunHost.AccessKeySecret)
	if err != nil {
		fmt.Println("Failed to create DNSManager:", err)
		return
	}

	//dp(domainParts) 将域名（qp[0]）分割为子域名与根域名，如 www:example.cn.eu.org => [www, example.cn.eu.org]
	dp := strings.Split(domain, ":")
	if len(dp) != 2 {
		fmt.Println("域名配置为需要修改的域名:主域名，比较要修改www.example.com，应该为www:example.com")
		return
	}
	err = manager.ManageSubDomain(dp[1], dp[0], "AAAA", ipv6)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
}

// Start server
func main() {
	loadConfig()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlePost(w, r)
		} else {
			handleRoot(w, r)
		}
	})
	http.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleListGet(w, r)
		} else if r.Method == http.MethodPost {
			handleListPost(w, r)
		}
	})

	log.Println("Server started: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

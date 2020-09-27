package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
)

func getCurrentIP() string {
	const GetTimeout = 5
	ipProvider := viper.GetString("ipProvider")

	req, err := http.NewRequest(http.MethodGet, ipProvider, nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("Content-Type", "plain/text; charset=utf-8")

	client := http.Client{
		Timeout: GetTimeout * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}

	currentIP := string(body)

	// Strip newlines from the string
	return strings.Replace(currentIP, "\n", "", -1)
}

func getUUID() string {
	// Get ZONE UUID from domain
	// GET /domains/<DOMAIN>

	const GetTimeout = 5
	gandiAPIEndpoint := viper.GetString("gandi_api_endpoint")
	gandiAPISecret := viper.GetString("gandi_api_secret")
	domain := viper.GetString("domain")

	url := gandiAPIEndpoint + "/domains/" + domain
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("Content-Type", "plain/text; charset=utf-8")
	req.Header.Set("X-Api-Key", gandiAPISecret)

	client := http.Client{
		Timeout: GetTimeout * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}
	//log.Println(string(body))

	//  JSON:
	//	{
	//		"zone_uuid": Str,
	//		"domain_keys_href": Str,
	//		"fqdn": Str,
	//		"zone_href": Str,
	//		"automatic_snapshots": Bool,
	//		"zone_records_href": Str,
	//		"domain_records_href": Str,
	//		"domain_href": Str
	//	}
	uuid := gjson.Get(string(body), "zone_uuid").Str

	return uuid
}

func getDNSIP(uuid string) string {
	// Get DNS record for the subdomain
	// GET /zones/<UUID>/records/<NAME>/<TYPE>

	const GetTimeout = 5
	gandiAPIEndpoint := viper.GetString("gandi_api_endpoint")
	gandiAPISecret := viper.GetString("gandi_api_secret")
	subdomain := viper.GetString("subdomain")

	url := gandiAPIEndpoint + "/zones/" + uuid + "/records/" + subdomain + "/A"
	//log.Println(url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("Content-Type", "plain/text; charset=utf-8")
	req.Header.Set("X-Api-Key", gandiAPISecret)

	client := http.Client{
		Timeout: GetTimeout * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}
	//log.Println(string(body))

	//  JSON:
	//  {
	//  	"rrset_type": Str
	//  	"rrset_ttl": Int
	//  	"rrset_name": Str
	//  	"rrset_href": Str
	//  	"rrset_values": [Str]
	//  }

	dnsIP := gjson.Get(string(body), "rrset_values.0").Str

	return dnsIP
}

func updateDNSRecords(uuid string, currentIP string) {
	// Update DNS record with new IP
	// PUT /zones/<UUID>/records/<NAME>/<TYPE>

	const GetTimeout = 5
	gandiAPIEndpoint := viper.GetString("gandi_api_endpoint")
	gandiAPISecret := viper.GetString("gandi_api_secret")
	ttl := viper.GetString("ttl")
	domain := viper.GetString("domain")
	subdomain := viper.GetString("subdomain")

	log.Println("Updating " + subdomain + "." + domain + " with IP " + currentIP)

	url := gandiAPIEndpoint + "/zones/" + uuid + "/records/" + subdomain + "/A"
	var jsonStr = `{"rrset_ttl": ` + ttl + `, "rrset_name": "` + subdomain + `", "rrset_values": ["` + currentIP + `"]}`
	//log.Println(jsonStr)
	var payload = []byte(jsonStr)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", gandiAPISecret)

	client := http.Client{
		Timeout: GetTimeout * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	log.Println(res.Status)
	//log.Println(res.StatusCode)
	//log.Println(res.Header)
	//body, _ := ioutil.ReadAll(res.Body)
	//fmt.Println("response Body:", string(body))
}

func main() {
	viper.SetConfigName("config") // name of config file without extension
	viper.SetConfigType("yaml")   // type of the config file, optional
	viper.AddConfigPath(".")      // look for config in the working directory
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}
	viper.AutomaticEnv() // Use env variables when available

	// infinite loop
	for {

		currentIP := getCurrentIP()
		//log.Println(currentIP)

		uuid := getUUID()
		//log.Println(uuid)

		dnsIP := getDNSIP(uuid)
		//log.Println(dnsIP)

		if currentIP == dnsIP {
			log.Println("Current IP is correct, no need to update: " + currentIP)
		} else {
			log.Println("IP has changed!")
			log.Println("Current IP: " + currentIP)
			log.Println("IP in DNS: " + dnsIP)
			updateDNSRecords(uuid, currentIP)
		}

		interval := viper.GetInt("interval")
		log.Println("Sleeping for " + strconv.Itoa(interval) + "s")
		time.Sleep(time.Duration(interval) * time.Second)
	}

	// TODO: Use arguments (verbose)
	// TODO: Support list of providers
	// TODO: Support multiple subdomains
}

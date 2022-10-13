package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
)

// A function to download the ICS file from a remote URL
func DownloadICSFile(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Could not download ICS file from URL %s", url)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	return string(body)
}

func ParseICSFileToEvents(str string) []*ics.VEvent {
	var events []*ics.VEvent
	calendar, err := ics.ParseCalendar(strings.NewReader(str))
	if err != nil {
		log.Fatal("Could not parse events from ICS file")
	}
	for _, event := range calendar.Events() {
		events = append(events, event)
	}
	return events
}

func CompareVEventsIsEqual(newEvents []*ics.VEvent, oldEvents []*ics.VEvent) bool {
	var newIDs []string
	var oldIDs []string
	for _, event := range newEvents {
		newIDs = append(newIDs, event.Id())
	}
	for _, event := range oldEvents {
		oldIDs = append(oldIDs, event.Id())
	}
	sort.Strings(newIDs)
	sort.Strings(oldIDs)
	return reflect.DeepEqual(newIDs, oldIDs)
}

func LoadICSFile(filename string) string {
	var file *os.File

	if _, err := os.Stat(filename); err == nil {
		file, _ = os.Open(filename)
	} else if errors.Is(err, os.ErrNotExist) {
		file, _ = os.Create(filename)
	}

	fileString, _ := ioutil.ReadAll(file)

	return string(fileString)
}

func WriteICSFile(filename string, body string) error {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		log.Fatal("Could not write file...")
	}
	file.WriteString(body)
	return nil
}

func getEnvVariable(variable string) (string, error) {
	if os.Getenv(variable) != "" {
		return os.Getenv(variable), nil
	}
	file, err := os.Open(".env")
	if err != nil {
		log.Fatal(".env file is not available")
		return "", err
	} else {
		str, _ := ioutil.ReadAll(file)
		lines := strings.Split(string(str), "\n")
		for _, line := range lines {
			name := strings.Split(line, "=")[0]
			value := strings.Split(line, "=")[1]
			if variable == name {
				return value, nil
			}
		}
	}
	return "", nil
}

func run() {
	fmt.Println("Syncing latest calendar events...")
	url, err := getEnvVariable("CALENDAR_URL")
	if err != nil {
		log.Fatalf("Fatal error: %s", err)
		panic("Fatal error occurred")
	}
	filename, err := getEnvVariable("ICS_FILENAME")
	if err != nil {
		log.Fatalf("Fatal error: %s", err)
		panic("Fatal error occurred")
	}
	newEvents := ParseICSFileToEvents(DownloadICSFile(url))
	oldEvents := ParseICSFileToEvents(LoadICSFile(filename))

	if CompareVEventsIsEqual(oldEvents, newEvents) {
	} else {
		calendar := ics.NewCalendar()
		for _, event := range newEvents {
			calendar.AddVEvent(event)
		}
		WriteICSFile(filename, calendar.Serialize())
	}
}

func serveICS(w http.ResponseWriter, req *http.Request) {
	filename, err := getEnvVariable("ICS_FILENAME")
	if err != nil {
		log.Fatalf("Fatal error: %s", err)
		panic("Fatal error occurred")
	}
	file := LoadICSFile(filename)
	fmt.Fprintf(w, file)
}

func serveHealthCheck(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "{'status':'OK'}")
}

func main() {

	runEveryMins, err := getEnvVariable("RUN_INTERVAL")
	if err != nil {
		runEveryMins = "5"
	}
	runInterval, _ := strconv.Atoi(runEveryMins)

	fmt.Printf("Starting scheduled task on %d mins interval..\n", runInterval)
	go func() {
		for {
			run()
			<-time.After(time.Duration(runInterval) * time.Minute)
		}
	}()

	http.HandleFunc("/", serveICS)
	http.HandleFunc("/health", serveHealthCheck)

	println("Starting web server...")
	http.ListenAndServe(":5000", nil)
}

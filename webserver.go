package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"html/template"
	
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

const checkTemplate = `
<html>
<head><title>Rancher {{.Title}}</title></head>
<body>
<b>{{.Level}}:</b> {{.Alarm}}<br>
</body>
</html>
`

var ALARM_LEVELS = map[int]string{
	0: "OK",
	1: "WARNING",
	2: "CRITICAL",
	3: "UNKNOWN",
}

type checkTemplateData struct {
	Title string
	Alarm string
	Level string
}

var webCheckConfig *CheckClientConfig

func webServer(listen string, ccc *CheckClientConfig) {
	router := mux.NewRouter().StrictSlash(true)

	webCheckConfig = ccc

	router.HandleFunc("/environments", webCheckEnvironments)
	router.HandleFunc("/hosts", webCheckHosts)
	router.HandleFunc("/stacks", webCheckStacks)
	router.HandleFunc("/services", webCheckServices)

	loggedRouter := handlers.LoggingHandler(os.Stdout, router)
	fmt.Println("Listening on", listen)
	log.Fatal(http.ListenAndServe(listen, loggedRouter))
}

func webCheckResults(level int, alarm, title string, w http.ResponseWriter, r *http.Request) {
	t_data := checkTemplateData{Title:title,Alarm:alarm,Level:ALARM_LEVELS[level]}
	t, _ := template.New("webpage").Parse(checkTemplate)
	
	w.Header().Set("X-Alarm-Level", fmt.Sprintf("%d", level))
	w.WriteHeader(http.StatusOK)
	_ = t.Execute(w, t_data)
}

func webCheckEnvironments(w http.ResponseWriter, r *http.Request) {
	level, alarm := checkEnvironments(webCheckConfig)
	webCheckResults(level, alarm, "environments", w, r)
}

func webCheckHosts(w http.ResponseWriter, r *http.Request) {
	level, alarm := checkHosts(webCheckConfig)
	webCheckResults(level, alarm, "hosts", w, r)
}

func webCheckStacks(w http.ResponseWriter, r *http.Request) {
	level, alarm := checkStacks(webCheckConfig)
	webCheckResults(level, alarm, "stacks", w, r)
}

func webCheckServices(w http.ResponseWriter, r *http.Request) {
	level, alarm := checkServices(webCheckConfig)
	webCheckResults(level, alarm, "services", w, r)
}

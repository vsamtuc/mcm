package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/vsamtuc/mcm/pkg/greet"
)

type navLink struct {
	Label    string
	Href     string
	HxGet    string
	HxTarget string
	HxSwap   string
}

type teamSummary struct {
	Course  string
	Ready   string
	Pending string
}

type indexPageData struct {
	Title            string
	BrandName        string
	BrandInitials    string
	BrandTagline     string
	Menu             []navLink
	UserLabel        string
	ActivityEndpoint string
	TeamSummaries    []teamSummary
}

// NewMux creates a new HTTP mux with health check endpoints.
// The schemaReady function is used to determine readiness status.
func NewMux(schemaReady func() bool) http.Handler {
	if schemaReady == nil {
		schemaReady = func() bool { return true }
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		renderTemplate(w, "index", defaultIndexPageData())
	})
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !schemaReady() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("schema not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(greet.Hello("world")))
	})
	mux.HandleFunc("/ui/activity", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		now := time.Now().Format(time.Kitchen)
		fmt.Fprintf(w, `<div class="space-y-3">
			<div class="alert alert-info">
				<span>%s · 14 new students joined CSC 482.</span>
			</div>
			<div class="alert alert-success">
				<span>%s · Team "Blue Orion" finalized roster.</span>
			</div>
		</div>`, now, now)
	})
	return mux
}

func defaultIndexPageData() indexPageData {
	return indexPageData{
		Title:            "Multiuser Cluster Manager",
		BrandName:        "MCM",
		BrandInitials:    "M",
		BrandTagline:     "Student cloud for team projects",
		UserLabel:        "session",
		ActivityEndpoint: "/ui/activity",
		Menu: []navLink{
			{Label: "Dashboard", Href: "/"},
			{Label: "Courses", Href: "#courses"},
			{Label: "Teams", Href: "#teams"},
			{Label: "Approvals", Href: "#approvals"},
		},
		TeamSummaries: []teamSummary{
			{Course: "CSC 482", Ready: "11", Pending: "2"},
			{Course: "CS 544", Ready: "9", Pending: "5"},
			{Course: "EE 201", Ready: "14", Pending: "1"},
		},
	}
}

package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vsamtuc/mcm/pkg/course"
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
func NewMux(schemaReady func() bool, courseSvc course.Service) http.Handler {
	if schemaReady == nil {
		schemaReady = func() bool { return true }
	}
	mux := http.NewServeMux()
	registerCourseRoutes(mux, courseSvc)
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

func registerCourseRoutes(mux *http.ServeMux, svc course.Service) {
	if svc == nil {
		panic("course service is nil")
	}
	h := &courseHandler{service: svc}
	mux.HandleFunc("/api/courses", h.collection)
	mux.HandleFunc("/api/courses/", h.item)
}

type courseHandler struct {
	service course.Service
}

func (h *courseHandler) collection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listCourses(w, r)
	case http.MethodPost:
		h.createCourse(w, r)
	default:
		w.Header().Set("Allow", "GET,POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *courseHandler) item(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(strings.TrimPrefix(r.URL.Path, "/api/courses/"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getCourse(w, r, id)
	case http.MethodPut:
		h.updateCourse(w, r, id)
	case http.MethodDelete:
		h.deleteCourse(w, r, id)
	default:
		w.Header().Set("Allow", "GET,PUT,DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *courseHandler) listCourses(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *courseHandler) getCourse(w http.ResponseWriter, r *http.Request, id int64) {
	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		handleCourseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *courseHandler) createCourse(w http.ResponseWriter, r *http.Request) {
	var input course.CreateCourseInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	created, err := h.service.Create(r.Context(), input)
	if err != nil {
		handleCourseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *courseHandler) updateCourse(w http.ResponseWriter, r *http.Request, id int64) {
	var input course.UpdateCourseInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	updated, err := h.service.Update(r.Context(), id, input)
	if err != nil {
		handleCourseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *courseHandler) deleteCourse(w http.ResponseWriter, r *http.Request, id int64) {
	if err := h.service.Delete(r.Context(), id); err != nil {
		handleCourseError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func parseID(raw string) (int64, error) {
	trimmed := strings.Trim(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return 0, fmt.Errorf("empty id")
	}
	return strconv.ParseInt(trimmed, 10, 64)
}

func handleCourseError(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, nil)
	case errors.Is(err, course.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusRequestTimeout, err.Error())
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

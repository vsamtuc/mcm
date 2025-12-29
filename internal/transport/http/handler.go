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

	"github.com/vsamtuc/mcm/pkg/application"
	"github.com/vsamtuc/mcm/pkg/auth"
	"github.com/vsamtuc/mcm/pkg/course"
	"github.com/vsamtuc/mcm/pkg/greet"
)

type navLink struct {
	Label        string
	Href         string
	HxGet        string
	HxTarget     string
	HxSwap       string
	AllowedRoles []string
}

type teamSummary struct {
	Course  string
	Ready   string
	Pending string
}

type indexPageData struct {
	Title             string
	BrandName         string
	BrandInitials     string
	BrandTagline      string
	Menu              []navLink
	UserLabel         string
	UserEmail         string
	UserAvatar        string
	UserAuthenticated bool
	ActivityEndpoint  string
	AuthLoginURL      string
	AuthLogoutURL     string
	ContentTemplate   string
	DevelMode         bool
	TeamSummaries     []teamSummary
}

type authRegistrar interface {
	register(mux *http.ServeMux)
}

func buildMenu(user *auth.User) []navLink {
	links := []navLink{
		{Label: "Dashboard", Href: "/"},
		{Label: "Courses", Href: "/courses", AllowedRoles: []string{"professor"}},
		{Label: "Enroll", Href: "/enroll", AllowedRoles: []string{"student"}},
		{Label: "Teams", Href: "#teams"},
		{Label: "Approvals", Href: "#approvals"},
	}
	visible := make([]navLink, 0, len(links))
	for _, link := range links {
		if linkVisible(link, user) {
			visible = append(visible, link)
		}
	}
	return visible
}

func linkVisible(link navLink, user *auth.User) bool {
	if len(link.AllowedRoles) == 0 {
		return true
	}
	if user == nil {
		return false
	}
	if isAdmin(*user) {
		return true
	}
	for _, role := range link.AllowedRoles {
		if userHasRole(*user, role) {
			return true
		}
	}
	return false
}

// NewMux creates a new HTTP mux with health check endpoints and authentication routes.
// The schemaReady function is used to determine readiness status.
func NewMux(schemaReady func() bool, appSvc application.Service, authCfg AuthConfig) (http.Handler, error) {
	if schemaReady == nil {
		schemaReady = func() bool { return true }
	}
	var authCtrl authRegistrar
	if authCfg.DevMode {
		authCtrl = newDevelAuthController(authCfg)
	} else {
		ctrl, err := newAuthController(context.Background(), authCfg)
		if err != nil {
			return nil, err
		}
		authCtrl = ctrl
	}
	mux := http.NewServeMux()
	registerCourseRoutes(mux, appSvc)
	authCtrl.register(mux)
	devMode := authCfg.DevMode
	uiCourses := &coursePageHandler{service: appSvc, devel: devMode}
	uiEnroll := &enrollPageHandler{service: appSvc, devel: devMode}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if _, ok := auth.UserFrom(r.Context()); !ok {
			http.Redirect(w, r, LoginPath, http.StatusFound)
			return
		}
		data := defaultIndexPageData(devMode)
		if user, ok := auth.UserFrom(r.Context()); ok {
			applyUserToIndex(&data, user)
		}
		renderTemplate(w, "index", data)
	})
	mux.HandleFunc("/courses", uiCourses.handleRoot)
	mux.HandleFunc("/courses/", uiCourses.handleRoutes)
	mux.HandleFunc("/enroll", uiEnroll.handleRoot)
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
	return mux, nil
}

func defaultIndexPageData(devMode bool) indexPageData {
	return indexPageData{
		Title:            "Multiuser Cluster Manager",
		BrandName:        "MCM",
		BrandInitials:    "M",
		BrandTagline:     "Student cloud for team projects",
		UserLabel:        "Sign in",
		AuthLoginURL:     LoginPath,
		AuthLogoutURL:    LogoutPath,
		ContentTemplate:  "index_content",
		DevelMode:        devMode,
		ActivityEndpoint: "/ui/activity",
		Menu:             buildMenu(nil),
		TeamSummaries: []teamSummary{
			{Course: "CSC 482", Ready: "11", Pending: "2"},
			{Course: "CS 544", Ready: "9", Pending: "5"},
			{Course: "EE 201", Ready: "14", Pending: "1"},
		},
	}
}

func applyUserToIndex(data *indexPageData, user auth.User) {
	if data == nil {
		return
	}
	data.UserAuthenticated = true
	data.UserLabel = userDisplayName(user)
	data.UserEmail = user.Email
	data.UserAvatar = userInitials(user)
	data.Menu = buildMenu(&user)
}

func userDisplayName(user auth.User) string {
	if user.Username != "" {
		return user.Username
	}
	if user.Email != "" {
		return user.Email
	}
	if user.Subject != "" {
		return user.Subject
	}
	return "Signed in"
}

func userInitials(user auth.User) string {
	name := user.Username
	if name == "" {
		name = user.Email
	}
	if name == "" {
		name = user.Subject
	}
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '.' || r == '_' || r == '-' || r == ' '
	})
	if len(parts) == 0 {
		return "U"
	}
	if len(parts) == 1 {
		runes := []rune(parts[0])
		if len(runes) >= 2 {
			return strings.ToUpper(string(runes[0:2]))
		}
		return strings.ToUpper(string(runes[0:1]))
	}
	first := firstRune(parts[0])
	second := firstRune(parts[1])
	if second == "" {
		return strings.ToUpper(first)
	}
	return strings.ToUpper(first + second)
}

func userHasRole(user auth.User, role string) bool {
	for _, r := range user.Roles {
		if strings.EqualFold(r, role) {
			return true
		}
	}
	return false
}

func isAdmin(user auth.User) bool {
	return userHasRole(user, "admin")
}

func isProfessor(user auth.User) bool {
	return isAdmin(user) || userHasRole(user, "professor")
}

func isStudent(user auth.User) bool {
	return isAdmin(user) || userHasRole(user, "student")
}

func firstRune(s string) string {
	for _, r := range s {
		return string(r)
	}
	return ""
}

func registerCourseRoutes(mux *http.ServeMux, svc application.Service) {
	if svc == nil {
		panic("course service is nil")
	}
	h := &courseHandler{service: svc}
	mux.HandleFunc("/api/courses", h.collection)
	mux.HandleFunc("/api/courses/", h.route)
}

type coursePageHandler struct {
	service application.Service
	devel   bool
}

type enrollPageHandler struct {
	service application.Service
	devel   bool
}

type coursesPageData struct {
	indexPageData
	Courses []course.Course
	Error   string
}

func (h *enrollPageHandler) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/enroll" {
		http.NotFound(w, r)
		return
	}
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		http.Redirect(w, r, LoginPath, http.StatusFound)
		return
	}
	if isProfessor(user) {
		http.Redirect(w, r, "/courses", http.StatusFound)
		return
	}
	if !isStudent(user) {
		writeError(w, http.StatusForbidden, "insufficient role")
		return
	}
	data := defaultIndexPageData(h.devel)
	data.Title = "Enroll in courses"
	data.ContentTemplate = "enroll_content"
	applyUserToIndex(&data, user)
	renderTemplate(w, "enroll", data)
}

func (h *coursePageHandler) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/courses" && r.URL.Path != "/courses/" {
		h.handleRoutes(w, r)
		return
	}
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		http.Redirect(w, r, LoginPath, http.StatusFound)
		return
	}
	if !isProfessor(user) {
		http.Redirect(w, r, "/enroll", http.StatusFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.renderCourses(w, r, "")
	case http.MethodPost:
		h.createCourse(w, r)
	default:
		w.Header().Set("Allow", "GET,POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *coursePageHandler) handleRoutes(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/courses/")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		http.Redirect(w, r, "/courses", http.StatusMovedPermanently)
		return
	}
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		http.Redirect(w, r, LoginPath, http.StatusFound)
		return
	}
	if !isProfessor(user) {
		http.Redirect(w, r, "/enroll", http.StatusFound)
		return
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 || parts[1] != "instructors" {
		http.NotFound(w, r)
		return
	}
	courseID, err := parseID(parts[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if len(parts) == 2 && r.Method == http.MethodPost {
		h.addInstructor(w, r, courseID)
		return
	}
	if len(parts) == 4 && parts[3] == "remove" && r.Method == http.MethodPost {
		profID, err := parseID(parts[2])
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid professor id")
			return
		}
		h.removeInstructor(w, r, courseID, profID)
		return
	}
	http.NotFound(w, r)
}

func (h *coursePageHandler) createCourse(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		http.Redirect(w, r, LoginPath, http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderCourses(w, r, "invalid form data")
		return
	}
	input := course.CreateCourseInput{
		Code:  strings.TrimSpace(r.FormValue("code")),
		Title: strings.TrimSpace(r.FormValue("title")),
		Term:  strings.TrimSpace(r.FormValue("term")),
	}
	ctx := auth.WithUser(r.Context(), user)
	if _, err := h.service.CreateCourse(ctx, input); err != nil {
		h.renderCourses(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/courses", http.StatusFound)
}

func (h *coursePageHandler) addInstructor(w http.ResponseWriter, r *http.Request, courseID int64) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		http.Redirect(w, r, LoginPath, http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderCourses(w, r, "invalid form data")
		return
	}
	profID, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("professor_id")), 10, 64)
	if err != nil || profID <= 0 {
		h.renderCourses(w, r, "invalid professor id")
		return
	}
	role := strings.TrimSpace(r.FormValue("role"))
	ctx := auth.WithUser(r.Context(), user)
	if _, err := h.service.AddCourseInstructor(ctx, courseID, course.Instructor{ProfessorID: profID, Role: role}); err != nil {
		h.renderCourses(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/courses", http.StatusFound)
}

func (h *coursePageHandler) removeInstructor(w http.ResponseWriter, r *http.Request, courseID, professorID int64) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		http.Redirect(w, r, LoginPath, http.StatusFound)
		return
	}
	ctx := auth.WithUser(r.Context(), user)
	if _, err := h.service.RemoveCourseInstructor(ctx, courseID, professorID); err != nil {
		h.renderCourses(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/courses", http.StatusFound)
}

func (h *coursePageHandler) renderCourses(w http.ResponseWriter, r *http.Request, errMsg string) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		http.Redirect(w, r, LoginPath, http.StatusFound)
		return
	}
	ctx := auth.WithUser(r.Context(), user)
	courses, err := h.service.ListCourses(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	data := coursesPageData{indexPageData: defaultIndexPageData(h.devel), Courses: courses, Error: errMsg}
	data.ContentTemplate = "courses_content"
	applyUserToIndex(&data.indexPageData, user)
	renderTemplate(w, "courses", data)
}

type courseHandler struct {
	service application.Service
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

func (h *courseHandler) route(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/courses/")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		writeError(w, http.StatusBadRequest, "invalid course path")
		return
	}
	parts := strings.Split(trimmed, "/")
	id, err := parseID(parts[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid course id")
		return
	}
	if len(parts) == 1 {
		h.courseItem(w, r, id)
		return
	}
	switch parts[1] {
	case "instructors":
		if len(parts) == 2 {
			h.instructorsCollection(w, r, id)
			return
		}
		if len(parts) == 3 {
			profID, err := parseID(parts[2])
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid professor id")
				return
			}
			h.instructorItem(w, r, id, profID)
			return
		}
	}
	writeError(w, http.StatusNotFound, "resource not found")
}

func (h *courseHandler) courseItem(w http.ResponseWriter, r *http.Request, id int64) {
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

func (h *courseHandler) instructorsCollection(w http.ResponseWriter, r *http.Request, courseID int64) {
	switch r.Method {
	case http.MethodPost:
		h.addCourseInstructor(w, r, courseID)
	default:
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *courseHandler) instructorItem(w http.ResponseWriter, r *http.Request, courseID, professorID int64) {
	switch r.Method {
	case http.MethodDelete:
		h.removeCourseInstructor(w, r, courseID, professorID)
	default:
		w.Header().Set("Allow", http.MethodDelete)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *courseHandler) listCourses(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListCourses(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *courseHandler) getCourse(w http.ResponseWriter, r *http.Request, id int64) {
	item, err := h.service.GetCourse(r.Context(), id)
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
	created, err := h.service.CreateCourse(r.Context(), input)
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
	updated, err := h.service.UpdateCourse(r.Context(), id, input)
	if err != nil {
		handleCourseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *courseHandler) deleteCourse(w http.ResponseWriter, r *http.Request, id int64) {
	if err := h.service.DeleteCourse(r.Context(), id); err != nil {
		handleCourseError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

type instructorPayload struct {
	ProfessorID int64  `json:"professor_id"`
	Role        string `json:"role"`
}

func (h *courseHandler) addCourseInstructor(w http.ResponseWriter, r *http.Request, courseID int64) {
	var payload instructorPayload
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	created, err := h.service.AddCourseInstructor(r.Context(), courseID, course.Instructor{
		ProfessorID: payload.ProfessorID,
		Role:        payload.Role,
	})
	if err != nil {
		handleCourseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, created)
}

func (h *courseHandler) removeCourseInstructor(w http.ResponseWriter, r *http.Request, courseID, professorID int64) {
	updated, err := h.service.RemoveCourseInstructor(r.Context(), courseID, professorID)
	if err != nil {
		handleCourseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
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
	case errors.Is(err, course.ErrUnauthenticated):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, course.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusRequestTimeout, err.Error())
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

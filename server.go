package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
)

var log *logrus.Logger

// trick to conver my functions to http.Handler
type appHandler func(http.ResponseWriter, *http.Request) error

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		log.Println("appHandler error: " + e.Error())
	}
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func main() {
	log = logrus.New()
	log.SetReportCaller(true)
	log.SetLevel(logrus.TraceLevel)
	log.Formatter = &logrus.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return "", fmt.Sprintf("%s:%d", filename, f.Line)
		},
	}

	port := getEnv("PORT", "3001")
	baseUrl := getEnv("BASEURL", "crutchdev")
	elastic := getEnv("ELASTIC", "http://10.130.0.21:9400")
	db_host := getEnv("DB_HOST", "10.130.0.13:5432")
	db_user := getEnv("DB_USER", "pguser")
	db_password := getEnv("DB_PASSWORD", "pgpassword")

	es, err := initElasticHelper(elastic)
	if err != nil {
		log.Fatalf("Failed to init Elastic connection: %v\n", err)
	}

	db, err := initDBHelper(db_host, db_user, db_password, "optima3_severstal")
	if err != nil {
		log.Fatalf("Failed to init DB connection: %v\n", err)
	}

	auth := initAuthMiddleware(es, db)
	methods := initMethodHandlers(auth, es, db)

	router := mux.NewRouter().StrictSlash(true)
	crutchAPI := router.PathPrefix("/" + baseUrl + "/api").Subrouter()
	crutchAPI.Use(auth.authMiddleware)
	crutchAPI.Methods("POST").Path("/searchProducts").Handler(appHandler(methods.searchProductsHandler))
	crutchAPI.Methods("POST").Path("/getCurrentUser").Handler(appHandler(methods.getCurrentUser))

	crutch := router.PathPrefix("/" + baseUrl).Subrouter()
	fs := wrapHandler(http.FileServer(http.Dir("./frontend/dist")), "/"+baseUrl) //warpHandler is used to handle history mode URLs
	crutch.PathPrefix("/").Handler(http.StripPrefix("/"+baseUrl, fs))

	http.Handle("/", WithLogging(router))

	log.Printf("Server listening on port %s, base url %s\n", port, baseUrl)
	log.Panic(
		http.ListenAndServe(":"+port, nil),
	)
}

type NotFoundRedirectRespWr struct {
	http.ResponseWriter // We embed http.ResponseWriter
	status              int
}

func (w *NotFoundRedirectRespWr) WriteHeader(status int) {
	w.status = status // Store the status for our own use
	if status != http.StatusNotFound {
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *NotFoundRedirectRespWr) Write(p []byte) (int, error) {
	if w.status != http.StatusNotFound {
		return w.ResponseWriter.Write(p)
	}
	return len(p), nil // Lie that we successfully written it
}

func wrapHandler(h http.Handler, baseUrl string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nfrw := &NotFoundRedirectRespWr{ResponseWriter: w}
		h.ServeHTTP(nfrw, r)
		if nfrw.status == 404 {
			log.Printf("Redirecting %s to %s", r.RequestURI, baseUrl)
			http.Redirect(w, r, baseUrl, http.StatusFound)
		}
	}
}

func WithLogging(h http.Handler) http.Handler {
	logFn := func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()

		uri := r.RequestURI
		method := r.Method
		h.ServeHTTP(rw, r) // serve the original request

		duration := time.Since(start)

		log.Info(method, " ", uri, " ", duration)
	}
	return http.HandlerFunc(logFn)
}

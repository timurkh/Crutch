package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
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
	standinUrl := getEnv("STANDINURL", "standindev")

	elastic := getEnv("ELASTIC", "http://10.130.0.21:9400")

	prodDBHost := getEnv("PROD_DB_HOST", "10.130.0.13:5432")
	prodDBUser := getEnv("PROD_DB_USER", "pguser")
	prodDBPswd := getEnv("PROD_DB_PASSWORD", "pgpassword")
	prodDBDtbs := getEnv("PROD_DB_DATABASE", "optima3_severstal")

	//	crutchDBHost := getEnv("CRUTCH_DB_HOST", "127.0.0.1:5432")
	//	crutchDBUser := getEnv("CRUTCH_DB_USER", "pguser")
	//	crutchDBPswd := getEnv("CRUTCH_DB_PASSWORD", "pgpassword")
	//	crutchDBDtbs := getEnv("CRUTCH_DB_DATABASE", "crutch")

	es, err := initElasticHelper(elastic)
	if err != nil {
		log.Fatalf("Failed to init Elastic connection: %v\n", err)
	}

	prodDB, err := initProdDBHelper(prodDBHost, prodDBUser, prodDBPswd, prodDBDtbs)
	if err != nil {
		log.Fatalf("Failed to init DB connection: %v\n", err)
	}

	//	crutchDB, err := initCrutchDBHelper(crutchDBHost, crutchDBUser, crutchDBPswd, crutchDBDtbs)
	//	if err != nil {
	//		log.Fatalf("Failed to init DB connection: %v\n", err)
	//	}

	auth := initAuthMiddleware(es, prodDB)
	methods := initMethodHandlers(auth, es, prodDB)

	router := mux.NewRouter().StrictSlash(true)

	crutchMethods := router.PathPrefix("/" + baseUrl + "/methods").Subrouter()
	crutchMethods.Use(auth.authMiddleware)
	crutchMethods.Methods("GET").Path("/counterparts").Handler(appHandler(methods.getCounterpartsHandler))
	crutchMethods.Methods("GET").Path("/counterparts/excel").Handler(appHandler(methods.getCounterpartsExcelHandler))
	crutchMethods.Methods("GET").Path("/products").Handler(appHandler(methods.searchProductsHandler))
	crutchMethods.Methods("GET").Path("/orders").Handler(appHandler(methods.getOrdersHandler))
	crutchMethods.Methods("GET").Path("/orders/excel").Handler(appHandler(methods.getOrdersExcelHandler))
	crutchMethods.Methods("GET").Path("/orders/csv").Handler(appHandler(methods.getOrdersCSVHandler))
	crutchMethods.Methods("GET").Path("/order/{orderId}").Handler(appHandler(methods.getOrderHandler))
	crutchMethods.Methods("GET").Path("/currentUser").Handler(appHandler(methods.getCurrentUser))

	crutch := router.PathPrefix("/" + baseUrl).Subrouter()
	fsCrutch := wrapHandler(http.FileServer(http.Dir("./frontend/dist")), "/"+baseUrl) //wrapHandler is used to handle history mode URLs
	crutch.PathPrefix("/").Handler(http.StripPrefix("/"+baseUrl, fsCrutch))

	standinAPI := router.PathPrefix("/" + standinUrl + "/methods").Subrouter()
	standinAPI.Use(auth.authMiddleware)
	standinAPI.Methods("GET").Path("/current-user").Handler(appHandler(methods.getCurrentUserSI))
	standinAPI.Methods("GET").Path("/cart-preview").Handler(appHandler(methods.getCartContent))
	standinAPI.Methods("GET").Path("/products").Handler(appHandler(methods.searchProductsHandler))

	standin := router.PathPrefix("/" + standinUrl).Subrouter()
	fsStandin := wrapHandler(http.FileServer(http.Dir("./standin/dist")), "/"+standinUrl)
	standin.PathPrefix("/").Handler(http.StripPrefix("/"+standinUrl, fsStandin))

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
		if !strings.HasPrefix(r.RequestURI, baseUrl+"/methods/") {
			nfrw := &NotFoundRedirectRespWr{ResponseWriter: w}
			h.ServeHTTP(nfrw, r)

			if nfrw.status == http.StatusNotFound {
				log.Info("Requested ", r.RequestURI, ", Path ", r.URL.Path, ", Replied ", nfrw.status)
				r.URL.Path = "/"
				w.Header().Set("Content-Type", "text/html")
				h.ServeHTTP(w, r)
			}
		} else {
			h.ServeHTTP(w, r)
		}
	}
}

type CustomRespWr struct {
	http.ResponseWriter // We embed http.ResponseWriter
	status              int
}

func (w *CustomRespWr) WriteHeader(status int) {
	w.status = status // Store the status for our own use
	w.ResponseWriter.WriteHeader(status)
}

func WithLogging(h http.Handler) http.Handler {
	logFn := func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()

		uri := r.RequestURI
		method := r.Method

		crw := &CustomRespWr{ResponseWriter: rw}
		h.ServeHTTP(crw, r) // serve the original request

		duration := time.Since(start)

		log.Info(crw.status, " ", method, " ", uri, " ", duration)
	}
	return http.HandlerFunc(logFn)
}

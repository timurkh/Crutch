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

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	"industrial.market/crutch/docs"
	_ "industrial.market/crutch/docs"
)

// @title Industrial.Market API
// @version 1.0

// @host industrial.market
// @BasePath /crutch/methods
// @securityDefinitions.basic BasicAuth

var log = logrus.New()

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

func initAuthMethodHandlers() (*MethodHandlers, *AuthMiddleware, error) {

	elastic := getEnv("ELASTIC", "http://10.130.0.21:9400")

	prodDBHost := getEnv("PROD_DB_HOST", "10.130.0.13:5432")
	prodDBUser := getEnv("PROD_DB_USER", "pguser")
	prodDBPswd := getEnv("PROD_DB_PASSWORD", "pgpassword")
	prodDBDtbs := getEnv("PROD_DB_DATABASE", "optima3_severstal")

	crutchDBHost := getEnv("CRUTCH_DB_HOST", "127.0.0.1:5432")
	crutchDBUser := getEnv("CRUTCH_DB_USER", "pguser")
	crutchDBPswd := getEnv("CRUTCH_DB_PASSWORD", "pgpassword")
	crutchDBDtbs := getEnv("CRUTCH_DB_DATABASE", "crutch")

	es, err := initElasticHelper(elastic)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to init Elastic connection: %v\n", err)
	}

	prodDB, err := initProdDBHelper(prodDBHost, prodDBUser, prodDBPswd, prodDBDtbs)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to init DB connection: %v\n", err)
	}

	crutchDB, err := initCrutchDBHelper(crutchDBHost, crutchDBUser, crutchDBPswd, crutchDBDtbs)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to init DB connection: %v\n", err)
	}

	auth := initAuthMiddleware(prodDB, crutchDB)
	methods := initMethodHandlers(es, prodDB, crutchDB)

	return methods, auth, nil
}

func main() {
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
	docs.SwaggerInfo.BasePath = "/" + baseUrl + "/methods"
	standinUrl := getEnv("STANDINURL", "standindev")

	methods, auth, err := initAuthMethodHandlers()

	if err != nil {
		log.Fatalf(err.Error())
	}

	router := mux.NewRouter().StrictSlash(true)
	CSRF := csrf.Protect(
		[]byte("dG3d563vyukewv%Yetrsbvsfd%WYfvs!"),
		csrf.SameSite(csrf.SameSiteStrictMode),
		csrf.Secure(true),
		csrf.HttpOnly(true),
	)

	router.Use(CSRF)

	crutchMethods := router.PathPrefix("/" + baseUrl + "/methods").Subrouter()
	crutchMethods.Use(auth.authMiddleware)
	crutchMethods.Methods("GET").Path("/counterparts").Handler(appHandler(methods.getCounterpartsHandler))
	crutchMethods.Methods("GET").Path("/counterparts/excel").Handler(appHandler(methods.getCounterpartsExcelHandler))
	crutchMethods.Methods("GET").Path("/products").Handler(appHandler(methods.searchProductsHandler))
	crutchMethods.Methods("GET").Path("/orders").Handler(appHandler(methods.getOrdersHandler))
	crutchMethods.Methods("GET").Path("/orders/excel").Handler(appHandler(methods.getOrdersExcelHandler))
	crutchMethods.Methods("GET").Path("/orders/{orderId}").Handler(appHandler(methods.getOrderHandler))
	crutchMethods.Methods("GET").Path("/currentUser").Handler(appHandler(methods.getCurrentUser))
	crutchMethods.Methods("GET").Path("/apiCredentials").Handler(appHandler(methods.getApiCredentialsHandler))
	crutchMethods.Methods("PUT").Path("/apiCredentials").Handler(appHandler(methods.putApiCredentialsHandler))

	crutch := router.PathPrefix("/" + baseUrl).Subrouter()

	crutch.Methods("GET").PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("https://industrial.market/"+baseUrl+"/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("#swagger-ui"),
		httpSwagger.UIConfig(map[string]string{"onComplete": `() => {
					var script = document.createElement('script');
					script.src = 'https://cdn.jsdelivr.net/npm/iframe-resizer@4.3.2/js/iframeResizer.contentWindow.min.js';
					document.head.appendChild(script);
				}`,
		}),
		// hack to change layout:
		httpSwagger.BeforeScript(`
				var script = document.createElement('script');
				script.src = 'https://cdn.jsdelivr.net/npm/iframe-resizer@4.3.2/js/iframeResizer.contentWindow.min.js';
				document.head.appendChild(script);

				const ui_ = SwaggerUIBundle({
					url: "\/`+baseUrl+`\/swagger\/doc.json",
					deepLinking: true,
					docExpansion: "full",
					dom_id: "#swagger-ui",
					validatorUrl: null,
					presets: [
						SwaggerUIBundle.presets.apis,
					],
					plugins: [
						SwaggerUIBundle.plugins.DownloadUrl
					],
					layout: "BaseLayout"
				})
				window.ui = ui_
				if (window.ui != null)
					return
			`),
	))

	fsCrutch := singlePageAppHandler(http.FileServer(http.Dir("./frontend/dist")), "/"+baseUrl) //wrapHandler is used to handle history mode URLs
	crutch.PathPrefix("/").Handler(http.StripPrefix("/"+baseUrl, fsCrutch))

	standinAPI := router.PathPrefix("/" + standinUrl + "/methods").Subrouter()
	standinAPI.Use(auth.authMiddleware)
	standinAPI.Methods("GET").Path("/current-user").Handler(appHandler(methods.getCurrentUserSI))
	standinAPI.Methods("GET").Path("/cart-preview").Handler(appHandler(methods.getCartContent))
	standinAPI.Methods("GET").Path("/products").Handler(appHandler(methods.searchProductsHandler))

	standin := router.PathPrefix("/" + standinUrl).Subrouter()
	fsStandin := singlePageAppHandler(http.FileServer(http.Dir("./standin/dist")), "/"+standinUrl)
	standin.PathPrefix("/").Handler(http.StripPrefix("/"+standinUrl, fsStandin))

	http.Handle("/", WithLogging(router))

	log.Printf("Server listening on port %s, base url %s\n", port, baseUrl)
	log.Panic(
		http.ListenAndServe(":"+port, router),
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

func singlePageAppHandler(h http.Handler, baseUrl string) http.HandlerFunc {
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

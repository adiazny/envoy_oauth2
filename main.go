package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"net/http/httputil"

	"flag"

	"github.com/gorilla/mux"
)

const (
	athleteEndpoint = "https://www.strava.com/api/v3/athlete"
	hmacKey         = "93Wg15rHSp6/Si5bH756OE6mAqL9ntX5DQ7ug5NgncE="
)

type contextKey string

const contextEventKey contextKey = "event"

type parsedData struct {
	AccessToken string `json:"access_token"`
	Subject     string `json:"sub,omitempty"`
}

func oauthMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
			return
		}

		fmt.Printf("Print Headers: %s\n", dump)

		hc, err := r.Cookie("OauthHMAC")
		if err != nil {
			fmt.Printf("Inside HMAC: %s\n", hc)
			http.Error(w, fmt.Sprint(err), http.StatusUnauthorized)
			return
		}

		expires, err := r.Cookie("OauthExpires")
		if err != nil {
			fmt.Printf("Inside OauthExpires: %s\n", expires)
			http.Error(w, fmt.Sprint(err), http.StatusUnauthorized)
			return
		}

		host := r.Host
		if host == "" {
			fmt.Printf("Inside Host: %s\n", host)
			http.Error(w, fmt.Sprint(err), http.StatusUnauthorized)
			return
		}

		bearerTokenCookie, err := r.Cookie("BearerToken")
		if err != nil {
			http.Error(w, "BearerToken not present", http.StatusUnauthorized)
			return
		}
		accessToken := bearerTokenCookie.Value

		// message := fmt.Sprintf("%s%s%s", host, expires.Value, accessToken)

		// hsh := hmac.New(sha256.New, []byte(hmacKey))
		// hsh.Write(([]byte(message)))

		// calculatedHMAC := base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(hsh.Sum(nil))))
		// if hc.Value != calculatedHMAC {
		// 	http.Error(w, "HMAC validation Failed", http.StatusUnauthorized)
		// 	return
		// }

		//optionally lookup who the user really is
		athlete, err := getAthlete(accessToken)
		if err != nil {
			http.Error(w, "getAthlete error", http.StatusUnauthorized)
			return
		}

		event := &parsedData{
			AccessToken: accessToken,
			Subject:     athlete.Username,
		}

		rctx := context.WithValue(r.Context(), contextEventKey, *event)
		h.ServeHTTP(w, r.WithContext(rctx))
	})
}

type AthleteInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func getAthlete(accessToken string) (AthleteInfo, error) {
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, athleteEndpoint, nil)
	if err != nil {
		fmt.Printf("Error creating HTTP request %s: %s", athleteEndpoint, err.Error())
		return AthleteInfo{}, err
	}

	bearer := fmt.Sprintf("Bearer %s", accessToken)
	req.Header.Add("Authorization", bearer)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making HTTP GET request to Strava /athlete: %v", err)
		return AthleteInfo{}, err
	}

	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body %v", err.Error())
		return AthleteInfo{}, err
	}

	var athlete AthleteInfo

	json.Unmarshal(b, &athlete)
	fmt.Println("Alan's AthleteInfo", athlete)

	return athlete, nil
}

func gethandler(w http.ResponseWriter, r *http.Request) {

	val := r.Context().Value(contextKey("event")).(parsedData)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "User %v logged in.", val)

}

func main() {

	flag.Parse()
	router := mux.NewRouter()
	router.Methods(http.MethodGet).Path("/").HandlerFunc(gethandler)

	server := &http.Server{
		Addr:    ":8082",
		Handler: oauthMiddleware(router),
	}
	//http2.ConfigureServer(server, &http2.Server{})
	fmt.Println("Starting Server...")
	err := server.ListenAndServe()
	// err := server.ListenAndServeTLS("certs/backend.crt", "certs/backend.key")
	fmt.Printf("Unable to start Server %v", err)

}

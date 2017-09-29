package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/icza/session"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/viper"
)

type dbFeedItem struct {
	UUID      string    `json:"uuid"`
	GUID      string    `json:"guid"`
	Title     string    `json:"title"`
	Link      string    `json:"link"`
	Author    string    `json:"author"`
	FeedTitle string    `json:"feedtitle"`
	FeedLink  string    `json:"feedlink"`
	Folder    string    `json:"folder"`
	Content   string    `json:"content"`
	Published time.Time `json:"published"`
	Read      bool      `json:"read"`
}

type dbFolderWithCount struct {
	Folder string `json:"folder"`
	Count  string `json:"unread"`
}

// StartHTTPD starts built in http server
func StartHTTPD() {
	session.Global.Close()
	session.Global = session.NewCookieManagerOptions(session.NewInMemStore(), &session.CookieMngrOptions{AllowHTTP: true})
	defer session.Global.Close()

	router := httprouter.New()

	router.GET("/", AuthenticationRequired(IndexHandler))
	router.GET("/api/feeds", AuthenticationRequired(FeedsHandler))
	router.GET("/api/folders", AuthenticationRequired(FoldersHandler))
	router.POST("/api/feeds/:uuid/read", AuthenticationRequired(ItemReadHandler))
	router.POST("/api/folders/:folder/read", AuthenticationRequired(FolderReadHandler))

	staticDir := fmt.Sprintf("%s/static", viper.GetString("web.installationpath"))
	router.ServeFiles("/static/*filepath", http.Dir(staticDir))

	hostport := fmt.Sprintf("%s:%d", viper.GetString("web.host"), viper.GetInt("web.port"))
	log.Printf("Starting web server on %s", hostport)
	log.Fatal(http.ListenAndServe(hostport, router))
}

func isAuthViaSession(r *http.Request) bool {
	sess := session.Get(r)
	if sess == nil {
		return false
	} else if sess.CAttr("UserName") == viper.GetString("web.username") {
		return true
	}

	return false
}

func isAuthenticated(r *http.Request) bool {
	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)

	if len(s) != 2 {
		return false
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return false
	}

	return pair[0] == viper.GetString("web.username") && pair[1] == viper.GetString("web.password")
}

func AuthenticationRequired(handler func(http.ResponseWriter, *http.Request, httprouter.Params)) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		if isAuthViaSession(r) {
			handler(w, r, ps)
		} else {
			if isAuthenticated(r) {
				sess := session.Get(r)

				if sess == nil {
					sess = session.NewSessionOptions(&session.SessOptions{
						CAttrs: map[string]interface{}{"UserName": viper.GetString("web.username")},
					})
				}

				if sess.New() {
					session.Add(sess, w)
				}

				handler(w, r, ps)
			} else {
				w.Header().Set("WWW-Authenticate", `Basic realm="feeds2imap"`)
				w.WriteHeader(401)
				w.Write([]byte("401 Unauthorized\n"))
			}
		}
	}
}

func respondWithJSON(w http.ResponseWriter, v interface{}) {
	bs, err := json.Marshal(&v)

	if err != nil {
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(bs)
}

type errorResponse struct {
	Error string `json:"error"`
}

func respondWithError(w http.ResponseWriter, responseError error) {
	var res errorResponse
	res.Error = responseError.Error()

	bs, err := json.Marshal(&res)
	log.Println(res)

	if err != nil {
		log.Println(err)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bs)
}

type webTemplatePayload struct {
	PocketEnabled bool
}

func IndexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fname := fmt.Sprintf("%s/templates/index.html", viper.GetString("web.installationpath"))
	t, err := template.New("index.html").Delims("[[", "]]").ParseFiles(fname)

	if err != nil {
		respondWithError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	payload := webTemplatePayload{PocketEnabled: viper.GetBool("web.pocket.enabled")}
	err = t.Execute(w, payload)

	if err != nil {
		respondWithError(w, err)
		return
	}
}

func FeedsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var items []dbFeedItem
	var args []interface{}
	var whereFilters []string
	var whereString string

	folder := r.FormValue("folder")
	unread := r.FormValue("unread") == "true"
	page, err := strconv.Atoi(r.FormValue("page"))

	if err != nil {
		log.Printf("Error parsing page argument: %s", err)
		page = 0
	}

	if len(folder) > 0 {
		whereFilters = append(whereFilters, "folder=?")
		args = append(args, folder)
	}

	if unread {
		whereFilters = append(whereFilters, "read=0")
	}

	if len(whereFilters) > 0 {
		whereString = fmt.Sprintf("WHERE %s", strings.Join(whereFilters, " AND "))
	}

	ppage := 20
	offset := page * ppage

	args = append(args, ppage, offset)

	query := fmt.Sprintf("SELECT * FROM feeds %s ORDER BY published_at DESC LIMIT ? OFFSET ?;", whereString)

	rows, err := db.Query(query, args...)
	if err != nil {
		respondWithError(w, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		i, err := ScanRowToItem(rows)

		if err != nil {
			respondWithError(w, err)
			return
		}

		items = append(items, i)
	}

	respondWithJSON(w, items)
}

func FoldersHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var folders []dbFolderWithCount

	rows, err := db.Query("SELECT folder, SUM( CASE WHEN read = 0 THEN 1 ELSE 0 END ) AS count FROM feeds GROUP BY folder;")
	if err != nil {
		respondWithError(w, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var folder dbFolderWithCount
		err := rows.Scan(&folder.Folder, &folder.Count)

		if err != nil {
			respondWithError(w, err)
			return
		}

		folders = append(folders, folder)
	}

	respondWithJSON(w, folders)
}

func ItemReadHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	uuid := ps.ByName("uuid")
	err := MarkAsReadInDBByID(uuid)

	if err != nil {
		respondWithError(w, err)
		return
	}
}

func FolderReadHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	folder := ps.ByName("folder")
	err := MarkAsReadInDBByFolder(folder)

	if err != nil {
		respondWithError(w, err)
		return
	}
}

package funcs

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

type CreateBook struct {
	Id       int    `json:"id"`
	Place_id int    `json:"place_id"`
	User_id  int    `json:"user_id"`
	From     string `json:"time_from"`
	To       string `json:"time_to"`
}

func GetPort() string { //готово, но можно переделать
	fmt.Println("Введите порт")
	portPtr := flag.String("port", "", "номер порта")
	flag.Parse()
	port := *portPtr
	port = strings.TrimSpace(port)
	if port == "" {
		fmt.Println("Порт пустой")
		err := errors.New("Введен не правильный порт")
		panic(err)
	}
	return port
}

func StartServer() error { //готово, но можно перепроверить
	port := GetPort()
	router := mux.NewRouter()
	router.Path("/ping").Methods("GET").HandlerFunc(HandlePing)
	router.Path("/book").Methods("POST").Queries("place_id", "user_id", "from", "to").HandlerFunc(HandleBook)
	router.Path("/booklist").Methods("GET").Queries("user_id").HandlerFunc(HandleBookList)
	router.Path("/booklist").Methods("GET").Queries("place_id").HandlerFunc(HandleBookList)

	return http.ListenAndServe(":"+port, router)
}

func HandlePing(w http.ResponseWriter, r *http.Request) { //доработать
	w.WriteHeader(http.StatusOK)
	response := map[string]string{"status": "ok"}
	json.NewEncoder(w).Encode(response)
}

func HandleBook(w http.ResponseWriter, r *http.Request) { // доработать
	query := r.URL.Query()
	place_id, _ := strconv.Atoi(query.Get("place_id"))
	user_id, _ := strconv.Atoi(query.Get("user_id"))
	NewBook := CreateBook{
		Place_id: place_id,
		User_id:  user_id,
		From:     query.Get("from"),
		To:       query.Get("to"),
	}
	ctx := context.Background()
	conn, err := CreateSQLConnetction()
	if err != nil {
		fmt.Println("Ошибка подключения к бд")
	}
	defer conn.Close(ctx)
	err = InsertBook(ctx, conn, NewBook)
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		return
	}
}

func HandleBookList(w http.ResponseWriter, r *http.Request) { //готово
	ctx := context.Background()
	conn, _ := CreateSQLConnetction()
	defer conn.Close(ctx)

	query := r.URL.Query()

	if UserIdStr := query.Get("user_id"); UserIdStr != "" {
		UserId, _ := strconv.Atoi(UserIdStr)
		SQLQuery := `
		SELECT id, user_id, place_id, time_from, time_to
		FROM bookings
		WHERE user_id = $1
		`

		transferData := []CreateBook{}

		rows, err := conn.Query(ctx, SQLQuery, UserId)
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		for rows.Next() {
			var b CreateBook
			rows.Scan(&b.Id,
				&b.User_id,
				&b.Place_id,
				&b.From,
				&b.To)
			transferData = append(transferData, b)
		}

		w.Header().Set("Content-type", "application/json")
		json.NewEncoder(w).Encode(transferData)

	}
	if PlaceIdStr := query.Get("place_id"); PlaceIdStr != "" {
		PlaseId, _ := strconv.Atoi(PlaceIdStr)
		ctx := context.Background()
		conn, _ := CreateSQLConnetction()
		SQLquery := `
		SELECT id, user_id, place_id, time_from, time_to
		FROM bookings
		WHERE place_id = $1
		`

		transferData := []CreateBook{}

		rows, err := conn.Query(ctx, SQLquery, PlaseId)
		if err != nil {
			panic(err)
		}
		defer rows.Close()

		for rows.Next() {
			var b CreateBook
			rows.Scan(&b.Id,
				&b.User_id,
				&b.Place_id,
				&b.From,
				&b.To)
			transferData = append(transferData, b)
		}

		w.Header().Set("Content-type", "application/json")
		json.NewEncoder(w).Encode(transferData)
	}
}

func CreateSQLConnetction() (*pgx.Conn, error) { //готово
	ctx := context.Background()
	host := os.Getenv("DB_HOST")
	DBport := os.Getenv("DB_PORT")
	DBuser := os.Getenv("DB_USER")
	DBpass := os.Getenv("DB_PASSWORD")
	DBname := os.Getenv("DB_NAME")
	Connectionstring := "postgres://" + DBuser + ":" + DBpass + "@" + host + ":" + DBport + "/" + DBname
	return pgx.Connect(ctx, Connectionstring)
}

func CreateTable(ctx context.Context, conn *pgx.Conn) { //готово
	SqlQuery := `
	CREATE TABLE IF NOT EXISTS bookings (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL,
		place_id INTEGER NOT NULL,
		time_from TIMESTAMP NOT NULL,
		time_to TIMESTAMP NOT NULL
	);
	`
	conn.Exec(ctx, SqlQuery)

}

func InsertBook(ctx context.Context, conn *pgx.Conn, nb CreateBook) error { //готово
	sqlQuery := `
	INSERT INTO bookings (place_id, user_id, time_from, time_to)
	SELECT $1, $2, $3, $4
	WHERE NOT EXISTS (
		SELECT 1 FROM bookings
		WHERE place_id = $1
		AND time_from < $4
		AND time_to > $3
	);
	`
	tag, err := conn.Exec(ctx, sqlQuery, nb.Place_id, nb.User_id, nb.From, nb.To)
	if tag.RowsAffected() == 0 {
		err := errors.New("Conflict")
		return err
	}
	return err
}

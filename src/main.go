package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4"
)

var db *pgx.Conn

func initDB() (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("POSTGRES_CONN"))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}
	log.Println("Successfully connected to the database!")
	return conn, nil
}

func PingHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("PingHandler started")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func main() {
	var err error
	log.Println("Program started")
	db, err = initDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(context.Background())
	router := mux.NewRouter()
	router.HandleFunc("/api/ping", PingHandler).Methods("GET")
	router.HandleFunc("/api/tenders", ShowTendersHandler).Methods("GET")
	router.HandleFunc("/api/tenders/new", CreateTenderHandler).Methods("POST")
	router.HandleFunc("/api/tenders/my", ShowUsersTendersHandler).Methods("GET")
	router.HandleFunc("/api/tenders/{tenderId}/status", ShowTenderStatusHandler).Methods("GET")
	router.HandleFunc("/api/tenders/{tenderId}/status", EditTenderStatusHandler).Methods("PUT")
	router.HandleFunc("/api/tenders/{tenderId}/edit", EditTenderHandler).Methods("PATCH")
	router.HandleFunc("/api/tenders/{tenderId}/rollback/{version}", TenderRollbackHandler).Methods("PUT")
	router.HandleFunc("/api/bids/new", CreateBidHandler).Methods("POST")
	router.HandleFunc("/api/bids/my", ShowUsersBidsHandler).Methods("GET")
	router.HandleFunc("/api/bids/{tenderId}/list", ShowTenderBidsHandler).Methods("GET")
	router.HandleFunc("/api/bids/{bidId}/status", ShowBidStatusHandler).Methods("GET")
	router.HandleFunc("/api/bids/{bidId}/status", EditBidStatusHandler).Methods("PUT")
	router.HandleFunc("/api/bids/{bidId}/edit", EditBidHandler).Methods("PATCH")
	router.HandleFunc("/api/bids/{bidId}/submit_decision", SubmitDecisionHandler).Methods("PUT")
	router.HandleFunc("/api/bids/{bidId}/rollback/{version}", BidRollbackHandler).Methods("PUT")
	router.HandleFunc("/api/bids/{bidId}/feedback", BidReviewHandler).Methods("PUT")
	router.HandleFunc("/api/bids/{tenderId}/reviews", ShowBidReviewsHandler).Methods("GET")

	server_address := os.Getenv("SERVER_ADDRESS")
	log.Printf("Starting server at %s\n", server_address)
	log.Fatal(http.ListenAndServe(server_address, router))
}

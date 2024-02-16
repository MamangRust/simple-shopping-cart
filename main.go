package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	db  *gorm.DB
	rdb *redis.Client
)

type Product struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	Name     string `json:"name"`
	Price    int    `json:"price"`
	Quantity int    `json:"quantity"`
}

type CartItem struct {
	ProductID uint `json:"product_id"`
	Quantity  int  `json:"quantity"`
	UserID    uint `json:"user_id"`
}

type Cart struct {
	UserID  uint       `json:"user_id" gorm:"primaryKey"`
	Items   []CartItem `json:"items" gorm:"-"`
	Updated time.Time  `json:"updated"`
}

func main() {
	// Initialize GORM database
	var err error
	db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	db.AutoMigrate(&Product{}, &Cart{})

	// Initialize Redis client
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// Initialize HTTP server
	r := mux.NewRouter()
	r.HandleFunc("/products", GetProductsHandler).Methods("GET")
	r.HandleFunc("/products", CreateProductHandler).Methods("POST")
	r.HandleFunc("/cart/{userID}", GetCartHandler).Methods("GET")
	r.HandleFunc("/cart/{userID}", AddToCartHandler).Methods("POST")
	r.HandleFunc("/cart/{userID}", DeleteCartHandler).Methods("DELETE")
	r.HandleFunc("/cart/{userID}/items", DeleteManyItemsHandler).Methods("DELETE")

	http.Handle("/", r)
	log.Println("Server is running on :8080...")
	http.ListenAndServe(":8080", nil)
}

func GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	var products []Product
	db.Find(&products)
	json.NewEncoder(w).Encode(products)
}

func CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	var product Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db.Create(&product)
	json.NewEncoder(w).Encode(product)
}

func GetCartHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	ctx := r.Context()
	cartJSON, err := rdb.Get(ctx, fmt.Sprintf("cart:%s", userID)).Result()
	if err == redis.Nil {
		json.NewEncoder(w).Encode([]CartItem{})
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var cart struct {
		UserID uint       `json:"user_id"`
		Items  []CartItem `json:"items"`
	}

	if err := json.Unmarshal([]byte(cartJSON), &cart); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(cart.Items)
}

func AddToCartHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	userInt, _ := strconv.Atoi(userID)

	var item CartItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	item.UserID = uint(userInt) // Set the UserID

	var cart Cart
	db.FirstOrCreate(&cart, Cart{UserID: item.UserID})
	cart.Items = append(cart.Items, item)
	cart.Updated = time.Now()
	db.Save(&cart)

	ctx := r.Context()
	cartJSON, _ := json.Marshal(cart)
	if err := rdb.Set(ctx, fmt.Sprintf("cart:%s", userID), cartJSON, 0).Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(cart)
}

func DeleteCartHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	ctx := r.Context()
	if err := rdb.Del(ctx, fmt.Sprintf("cart:%s", userID)).Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Cart deleted successfully"})
}

func DeleteManyItemsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	var items []CartItem
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	cartJSON, err := rdb.Get(ctx, fmt.Sprintf("cart:%s", userID)).Result()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var cart []CartItem
	if err := json.Unmarshal([]byte(cartJSON), &cart); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, item := range items {
		for i, cartItem := range cart {
			if cartItem.ProductID == item.ProductID {
				cart = append(cart[:i], cart[i+1:]...)
				break
			}
		}
	}

	updatedCartJSON, err := json.Marshal(cart)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := rdb.Set(ctx, fmt.Sprintf("cart:%s", userID), updatedCartJSON, 0).Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Items deleted successfully"})
}

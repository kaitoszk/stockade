package handler

import "net/http"

// NewRouterはURLとhandlerの対応表を作る
func NewRouter(products *ProductHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /products", products.Create)
	return mux
}
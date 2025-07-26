package main

import (
	"html/template"
	"log"
	"net/http"
)

// Универсальный рендер страниц
func render(w http.ResponseWriter, page string, data interface{}) {
	ts, err := template.ParseFiles("./templates/base.layout.tmpl", page)
	if err != nil {
		log.Println("Ошибка парсинга шаблона:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = ts.Execute(w, data)
	if err != nil {
		log.Println("Ошибка выполнения шаблона:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

package shared

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

func ParseID(path, prefix string) (int, error) {
	idStr := strings.TrimPrefix(path, prefix)
	parts := strings.Split(idStr, "/")
	return strconv.Atoi(parts[0])
}
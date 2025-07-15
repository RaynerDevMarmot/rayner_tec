package main

import (
	"database/sql" // Para la conexión a la base de datos
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os" // Para leer variables de entorno

	_ "github.com/go-sql-driver/mysql" // <--- Driver para MySQL
)

// Solicitud representa la estructura de los datos que recibiremos del formulario
type Solicitud struct {
	Nombre   string `json:"nombre"`
	Telefono string `json:"telefono"`
	Servicio string `json:"servicio"`
}

// Global variable for the database connection (for simplicity in this example)
var db *sql.DB

func main() {
	// --- Configuración de la Base de Datos (MySQL en este ejemplo) ---
	// Railway inyecta la URL de la base de datos en una variable de entorno.
	// Para MySQL en Railway, la variable de entorno es normalmente MYSQL_URL.
	dbURL := os.Getenv("MYSQL_URL") // <--- Usamos MYSQL_URL para Railway
	if dbURL == "" {
		log.Fatal("La variable de entorno MYSQL_URL no está configurada. Asegúrate de que Railway la esté inyectando o configúrala localmente para pruebas.")
	}

	var err error
	// Abre la conexión a la base de datos
	db, err = sql.Open("mysql", dbURL) // <--- Conector "mysql"

	if err != nil {
		log.Fatalf("Error al conectar a la base de datos: %v", err)
	}
	defer db.Close() // Asegúrate de cerrar la conexión cuando la aplicación se detenga

	// Prueba la conexión
	err = db.Ping()
	if err != nil {
		log.Fatalf("Error al hacer ping a la base de datos: %v", err)
	}
	fmt.Println("Conexión a la base de datos MySQL establecida con éxito.")

	// --- Crear la tabla si no existe (solo si es la primera vez) ---
	// Adapta la consulta SQL para MySQL.
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS solicitudes (
		id INT AUTO_INCREMENT PRIMARY KEY,
		nombre VARCHAR(255) NOT NULL,
		telefono VARCHAR(255) NOT NULL,
		servicio VARCHAR(255) NOT NULL,
		fecha_creacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	` // <--- Consulta SQL para MySQL
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Error al crear la tabla 'solicitudes': %v", err)
	}
	fmt.Println("Tabla 'solicitudes' verificada/creada con éxito.")

	// --- Configuración de la API ---
	// La ruta principal se configura para manejar CORS y redirigir
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Permitir cualquier origen (¡CUIDADO EN PRODUCCIÓN!)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")

		// Manejar pre-flight requests (OPTIONS)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Si la ruta es el endpoint de envío, pasamos al handler específico
		if r.URL.Path == "/submit-service" {
			submitServiceHandler(w, r)
			return
		}

		// Si es cualquier otra ruta, mostramos un mensaje por defecto
		http.Error(w, "Bienvenido a la API de servicios. Usa /submit-service para enviar datos.", http.StatusOK)
	})

	// Obtener el puerto del entorno (Railway lo inyecta en PORT)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Puerto por defecto para desarrollo local
	}

	fmt.Printf("Servidor Go escuchando en el puerto :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func submitServiceHandler(w http.ResponseWriter, r *http.Request) {
	// Configurar CORS para esta respuesta específica también
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, `{"message": "Método no permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	var solicitud Solicitud
	err := json.NewDecoder(r.Body).Decode(&solicitud)
	if err != nil {
		http.Error(w, `{"message": "Error al decodificar la solicitud JSON"}`, http.StatusBadRequest)
		return
	}

	log.Printf("Solicitud recibida para el servicio '%s': Nombre='%s', Teléfono='%s'", solicitud.Servicio, solicitud.Nombre, solicitud.Telefono)

	// --- Insertar en la base de datos ---
	// Adapta la consulta SQL para MySQL con marcadores de posición "?"
	insertSQL := `INSERT INTO solicitudes (nombre, telefono, servicio) VALUES (?, ?, ?)` // <--- Consulta SQL para MySQL
	_, err = db.Exec(insertSQL, solicitud.Nombre, solicitud.Telefono, solicitud.Servicio)
	if err != nil {
		log.Printf("Error al insertar en la base de datos: %v", err)
		http.Error(w, `{"message": "Error interno del servidor al guardar la solicitud"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Solicitud recibida con éxito!"})
}

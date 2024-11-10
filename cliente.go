// Cliente
package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	// Conectar al servidor
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error al conectar al servidor:", err)
		return
	}
	defer conn.Close()

	// Leer la respuesta del servidor
	reader := bufio.NewReader(conn)
	line, _ := reader.ReadString('\n')
	fmt.Print(line) // Bienvenida

	// Ingresar el userID
	fmt.Print("Ingresa tu ID de usuario: ")
	var userID int
	fmt.Scanln(&userID)
	fmt.Fprintln(conn, userID)

	// Leer la respuesta del servidor después de ingresar el ID
	line, _ = reader.ReadString('\n')
	fmt.Print(line) // "Archivos CSV leídos correctamente"

	// Leer y mostrar los géneros disponibles
	line, _ = reader.ReadString('\n')
	fmt.Print(line) // Mostrar la instrucción de elegir un género

	// Leer la lista completa de géneros
	var genres []string
	for {
		line, _ = reader.ReadString('\n')
		if line == "[END_OF_GENRES]\n" { // Fin de la lista de géneros
			break
		}
		fmt.Print(line) // Mostrar géneros
		genres = append(genres, line)
	}

	// Seleccionar un género
	var genreIndex int
	fmt.Print("Por favor, selecciona un género por el número: ")
	fmt.Scanln(&genreIndex)

	// Enviar la selección al servidor
	fmt.Fprintln(conn, genreIndex)

	// Leer y mostrar las películas recomendadas con calificación
	fmt.Println("Películas recomendadas:")
	for {
		line, _ = reader.ReadString('\n')
		if line == "\n" { // Fin de las recomendaciones
			break
		}
		fmt.Print(line) // Mostrar películas con calificación promedio
	}
}

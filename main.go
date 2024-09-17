package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
)

/*
1º ver returns de tudo crl erros panics
tratar de docker images, nao é preciso criar a imagem se ja existir

programar para evitar racetime conditions

FALTA CHECKAR PORTAS MISTURADAS DO GENERO PORTAS DO SV COM HAPOXY
*/

const HAPROXYPORT = "8080"
const WEBSVPORT = ":9090"

func getAllServersInfo(w http.ResponseWriter, r *http.Request) {
	dockerInfo := getAllDockerInfoJson()

	//convert to json
	json, err := json.Marshal(dockerInfo)
	if err != nil {
		fmt.Println(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func getRandomServerInt() string {
	randS, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		panic(err)
	}
	return "s" + randS.String()
}

func checkUserInput(serverName string, serverPort string, dockerPassword string, dockerImageName string) bool {
	if serverName == "" || serverPort == "" || dockerPassword == "" || dockerImageName == "" {
		return false
	}

	serverPortInt, err := strconv.Atoi(serverPort)
	if err != nil {
		return false
	}

	if serverPortInt < 0 || serverPortInt > 65535 {
		return false
	}

	return true
}

// ver racetime nas condicoes
func handleCreateNewDockerServer(w http.ResponseWriter, r *http.Request) {
	//get the server name
	serverName := r.FormValue("serverName")
	dockerPassword := r.FormValue("dockerPassword")

	serverName = "vmForge_" + serverName

	serverPort, err := generatePort()
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	infoOK := checkUserInput(serverName, serverPort, dockerPassword, "vm_forge_minimal")
	if !infoOK {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = runDocker(dockerPassword, serverPort, "vm_forge_minimal", serverName)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	err = createNewHAPROXYServer(serverName, getRandomServerInt(), serverPort)
	if err != nil {
		stopContainer(serverName)
		deleteContainer(serverName)
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	restartHaProxy(HAPROXYPORT)

	w.WriteHeader(http.StatusOK)
}

func stopServer(w http.ResponseWriter, r *http.Request) {
	containerID := r.FormValue("containerID")
	fmt.Printf("Stopping container with ID: %s\n", containerID)

	err := stopContainer(containerID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func startServer(w http.ResponseWriter, r *http.Request) {
	containerID := r.FormValue("containerID")
	fmt.Printf("Starting container with ID: %s\n", containerID)

	err := startContainer(containerID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func restartServer(w http.ResponseWriter, r *http.Request) {
	containerID := r.FormValue("containerID")
	fmt.Printf("Restarting container with ID: %s\n", containerID)

	err := restartContainer(containerID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func deleteServer(w http.ResponseWriter, r *http.Request) {
	containerID := r.FormValue("containerID")
	fmt.Printf("Deleting container with ID: %s\n", containerID)

	serverName := findServerName(containerID)
	err := deleteHAPROXYServer(serverName)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	err = deleteContainer(containerID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	restartHaProxy(HAPROXYPORT)

	w.WriteHeader(http.StatusOK)
}

func handler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "website/html.html")
}

func handlerHaProxy(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "website/haproxy.html")
}

func handlerDockerController(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "website/dockercontainers.html")
}

func runWebsite() {
	http.HandleFunc("/handleCreateNewDockerServer", handleCreateNewDockerServer)
	http.HandleFunc("/getAllServersInfo", getAllServersInfo)
	http.HandleFunc("/stopServer", stopServer)
	http.HandleFunc("/startServer", startServer)
	http.HandleFunc("/restartServer", restartServer)
	http.HandleFunc("/deleteServer", deleteServer)

	http.HandleFunc("/", handler)
	http.HandleFunc("/haproxy-controller", handlerHaProxy)
	http.HandleFunc("/docker-containers", handlerDockerController)

	fmt.Println("Running website on port", WEBSVPORT)
	log.Fatal(http.ListenAndServe(WEBSVPORT, nil))
}

func main() {
	built := checkIfDockerNameIsBuilt("vm_forge_minimal")

	if !built {
		path := createFolder(DOCKER_FILE_FOLDER_NAME)
		getDockerFiles(path, "temp123") // temporary way to get dockerFiles so need to change this

		err := buildDockerImage(path, "vm_forge_minimal")
		if err != nil {
			fmt.Println("runDocker: ", err)
			return
		}
	}

	built2 := checkIfDockerNameIsBuilt("vm_forge_minimal")

	if !built2 {
		fmt.Println("Error building docker image")
		return
	}

	err := runHaProxy(HAPROXYPORT)
	if err != nil {
		fmt.Println(err)
		return
	}

	runWebsite()

}

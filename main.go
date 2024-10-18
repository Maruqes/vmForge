package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"time"
)

/*
1º ver returns de tudo crl erros panics
tratar de docker images, nao é preciso criar a imagem se ja existir

programar para evitar racetime conditions

FALTA CHECKAR PORTAS MISTURADAS DO GENERO PORTAS DO SV COM HAPOXY
*/

const HAPROXYPORT = "8080"
const WEBSVPORT = ":9090"
const SERVER_LINK = "localhost"

var auth = Auth{}

type DockerConstInfo struct {
	ImageName string `json:"imageName"`
	Path      string `json:"path"`
}

var dockerConstInfo = []DockerConstInfo{
	{"vm_forge_minimal", "testData/DockerFileExample"},
	{"vm_forge_min_java", "testData/DockerFileJavaExample"},
}

func getAllServersInfo(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

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

func checkUserInput(serverName string, serverPort string, dockerPassword string, dockerImageName string) error {
	if serverName == "" || serverPort == "" || dockerPassword == "" || dockerImageName == "" {
		return fmt.Errorf("empty fields")
	}

	if !isValidInput(serverName) || !isValidInput(serverPort) || !isValidInput(dockerPassword) || !isValidInput(dockerImageName) {
		return fmt.Errorf("fields contain wierd characters")
	}

	serverPortInt, err := strconv.Atoi(serverPort)
	if err != nil {
		return fmt.Errorf("port is not a number")
	}

	if serverPortInt < 0 || serverPortInt > 65535 {
		return fmt.Errorf("port is invalid")
	}

	return nil
}

func isValidInput(s string) bool {
	for _, r := range s {
		if !(r >= '!' && r <= '~') {
			return false
		}
	}
	return true
}

func readUserCookies(r *http.Request) (string, string) {
	username := ""
	token := ""

	cookie, err := r.Cookie("username")
	if err == nil {
		username = cookie.Value
	}

	cookie, err = r.Cookie("token")
	if err == nil {
		token = cookie.Value
	}

	return username, token
}

func redirectToLoginPage(w http.ResponseWriter) {
	fmt.Fprint(w, `
			<script type="text/javascript">
				window.location.href = '/login';
			</script>
		`)
}

func redirectToMainPage(w http.ResponseWriter) {
	fmt.Fprint(w, `
			<script type="text/javascript">
				window.location.href = '/';
			</script>
		`)
}

func handleCreateNewDockerServer(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	//get the server name
	serverName := r.FormValue("serverName")
	dockerPassword := r.FormValue("dockerPassword")
	serverExample := r.FormValue("serverExample")

	dockerImage := "null"
	//check from static images
	for i := 0; i < len(dockerConstInfo); i++ {
		if dockerConstInfo[i].ImageName == serverExample {
			dockerImage = dockerConstInfo[i].ImageName
			break
		}
	}

	//check from db images
	sqlImagesNames := getDockerImagesNames()
	for i := 0; i < len(sqlImagesNames); i++ {
		if sqlImagesNames[i] == serverExample {
			dockerImage = serverExample
			break
		}
	}

	if dockerImage == "null" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	serverName = "vmForge_" + serverName

	serverPort, err := generatePort()
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	fmt.Printf("Creating new server with name: %s, port: %s, dockerPassword: %s, dockerImage: %s\n", serverName, serverPort, dockerPassword, dockerImage)

	infoOK := checkUserInput(serverName, serverPort, dockerPassword, dockerImage)
	if infoOK != nil {
		fmt.Println(infoOK)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(infoOK.Error()))
		return
	}

	err = runDocker(dockerPassword, serverPort, dockerImage, serverName)
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
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	containerID := r.FormValue("containerID")
	fmt.Printf("Stopping container with ID: %s\n", containerID)

	err = stopContainer(containerID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func startServer(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	containerID := r.FormValue("containerID")
	fmt.Printf("Starting container with ID: %s\n", containerID)

	err = startContainer(containerID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func restartServer(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	containerID := r.FormValue("containerID")
	fmt.Printf("Restarting container with ID: %s\n", containerID)

	err = restartContainer(containerID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func deleteServer(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	containerID := r.FormValue("containerID")
	fmt.Printf("Deleting container with ID: %s\n", containerID)

	serverName := findServerName(containerID)
	err = deleteHAPROXYServer(serverName)
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
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}
	http.ServeFile(w, r, "website/html.html")
}

func admin_page(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}
	http.ServeFile(w, r, "website/admin_page.html")
}

func handlerDockerController(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	http.ServeFile(w, r, "website/dockercontainers.html")
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, _ := auth.loginWithWebToken(username, token)

	if loginIn {
		w.WriteHeader(http.StatusOK)
		redirectToMainPage(w)
		return
	}

	http.ServeFile(w, r, "website/login.html")
}

func SetCookies(w http.ResponseWriter, username string, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "username",
		Value:    username,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "login_expiration_date",
		Value:    time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})
}

func loginUser(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	token, err := auth.login(username, password)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}

	SetCookies(w, username, token)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func refreshCookies(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	newToken, err := auth.refreshWebToken(username, token)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	SetCookies(w, username, newToken)

	redirectToMainPage(w)
	fmt.Println("Cookies refreshed for user: " + username)
}

func getImageNames(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	dockerConstInfoCopy := dockerConstInfo

	//convert to json
	jsonret, err := json.Marshal(dockerConstInfoCopy)
	if err != nil {
		fmt.Println(err)
	}

	dbImages := getDockerImagesNames()
	dockerInfoSql := DockerConstInfo{}
	for i := 0; i < len(dbImages); i++ {
		dockerInfoSql.ImageName = dbImages[i]
		dockerInfoSql.Path = "null"
		dockerConstInfoCopy = append(dockerConstInfoCopy, dockerInfoSql)
	}

	jsonret, err = json.Marshal(dockerConstInfoCopy)
	if err != nil {
		fmt.Println(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonret)
}

func getDockerPort(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(HAPROXYPORT))
}

func css(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "website/css.css")
}

func getDockerImages(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	images := getDockerImagesJson()

	//convert to json
	json, err := json.Marshal(images)
	if err != nil {
		fmt.Println(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func createDockerImageRequest(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	imageName := r.FormValue("imageName")
	commands := r.FormValue("commands")

	if imageName == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("imageName is empty"))
		return
	}
	imageName = "vm_forge_" + imageName

	commandsJson := []string{}
	err = json.Unmarshal([]byte(commands), &commandsJson)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("commands are not in the correct format"))
		return
	}

	err = createDockerImage(imageName, commandsJson)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func removeDockerImage(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	imageID := r.FormValue("imageID")
	if imageID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("imageID is empty"))
		return
	}

	imageName := r.FormValue("imageName")
	if imageName == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("imageName is empty"))
		return
	}

	err = removeDockerImageSQL(imageName)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	err = removeDockerImageCommand(imageID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getAdmins(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	admins := auth.getAdminsArrName()

	//convert to json
	json, err := json.Marshal(admins)
	if err != nil {
		fmt.Println(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func removeAdmin(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	adminName := r.FormValue("adminName")
	if adminName == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("adminName is empty"))
		return
	}
	if adminName == username {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("You can't remove yourself"))
		return
	}

	err = auth.removeUser(adminName)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	fmt.Println("Admin removed: " + adminName)
	w.WriteHeader(http.StatusOK)
}

func createAdmin(w http.ResponseWriter, r *http.Request) {
	username, token := readUserCookies(r)
	loginIn, err := auth.loginWithWebToken(username, token)
	if err != nil || !loginIn {
		w.WriteHeader(http.StatusUnauthorized)
		redirectToLoginPage(w)
		return
	}

	adminName := r.FormValue("adminName")
	if adminName == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("adminName is empty"))
		return
	}

	adminPassword := r.FormValue("adminPassword")
	if adminPassword == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("adminPassword is empty"))
		return
	}

	err = auth.register(adminName, adminPassword)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func openPortRequest(w http.ResponseWriter, r *http.Request) {
	serverName := r.FormValue("serverName")
	if serverName == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("serverName is empty"))
		return
	}

	port := r.FormValue("port")
	if port == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("port is empty"))
		return
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("port is not a number"))
		return
	}

	if portInt < 0 || portInt > 65535 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("port is invalid"))
		return
	}

	err = openPort(serverName, port)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func runWebsite() {
	//docker haproxy
	http.HandleFunc("/getDockerPort", getDockerPort)
	http.HandleFunc("/getImageNames", getImageNames)
	http.HandleFunc("/handleCreateNewDockerServer", handleCreateNewDockerServer)
	http.HandleFunc("/getAllServersInfo", getAllServersInfo)
	http.HandleFunc("/stopServer", stopServer)
	http.HandleFunc("/startServer", startServer)
	http.HandleFunc("/restartServer", restartServer)
	http.HandleFunc("/deleteServer", deleteServer)

	//docker images
	http.HandleFunc("/getDockerImages", getDockerImages)
	http.HandleFunc("/createDockerImage", createDockerImageRequest)
	http.HandleFunc("/removeDockerImage", removeDockerImage)

	//admins
	http.HandleFunc("/getAdmins", getAdmins)
	http.HandleFunc("/removeAdmin", removeAdmin)
	http.HandleFunc("/createAdmin", createAdmin)

	//website
	http.HandleFunc("/", handler)
	http.HandleFunc("/admin_page", admin_page)
	http.HandleFunc("/docker-containers", handlerDockerController)
	http.HandleFunc("/login", loginPage)
	http.HandleFunc("/css", css)

	//auth
	http.HandleFunc("/loginUser", loginUser)
	http.HandleFunc("/refreshCookie", refreshCookies)

	//ports
	http.HandleFunc("/portOpen", openPortRequest)

	fmt.Println("Running website on port", WEBSVPORT)
	log.Fatal(http.ListenAndServe(WEBSVPORT, nil))
}

func main() {
	auth.init()
	dockerInit()

	adminsNames := auth.getAdminsArrName()
	if len(adminsNames) == 0 {
		for i := 0; i < 5; i++ {
			fmt.Println("No admins found, creating default admin\n YOU SHOULD DELETE THIS ADMIN AFTER CREATING A NEW ONE ON THE PAGE\n username: admin, password: admin")
		}
		err := auth.register("admin", "admin")
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	for i := 0; i < len(dockerConstInfo); i++ {
		built0 := checkIfDockerImageIsBuilt(dockerConstInfo[i].ImageName, dockerConstInfo[i].Path)
		if !built0 {
			fmt.Printf("Error building docker image on image %s\n", dockerConstInfo[i].ImageName)
			return
		}
	}

	err := runHaProxy(HAPROXYPORT)
	if err != nil {
		fmt.Println(err)
		return
	}

	runWebsite()
}

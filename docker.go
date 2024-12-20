package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/exp/rand"
)

const DOCKER_FILE_FOLDER_NAME = "/dockerFiles"

func check_ports(ports []haproxyPorts) error {
	for i := 0; i < len(ports); i++ {
		if ports[i].ServerPort == "" {
			return fmt.Errorf("port is empty")
		}
		portInt, err := strconv.Atoi(ports[i].ServerPort)
		if err != nil {
			return err
		}
		if portInt < 0 || portInt > 65535 {
			return fmt.Errorf("port is invalid")
		}
		use := isPortInUse(portInt)
		if use {
			return fmt.Errorf("port is already in use")
		}

		if ports[i].ServerName == "" {
			return fmt.Errorf("server name is empty")
		}

		if ports[i].Subdomain == "" {
			return fmt.Errorf("Subdomain is empty")
		}

		for j := 0; j < len(ports); j++ {
			if i != j && ports[i].ServerPort == ports[j].ServerPort {
				return fmt.Errorf("port is already in use")
			}

			if i != j && ports[i].Subdomain == ports[j].Subdomain {
				return fmt.Errorf("Subdomain is already in use")
			}
		}

		//checking Subdomains
		if ports[i].Subdomain[0] == '-' || ports[i].Subdomain[len(ports[i].Subdomain)-1] == '-' {
			return fmt.Errorf("Subdomain is invalid (starts or ends with -)")
		}

		for j := 0; j < len(ports[i].Subdomain); j++ {
			if ports[i].Subdomain[j] != '-' && !((ports[i].Subdomain[j] >= 'a' && ports[i].Subdomain[j] <= 'z') || (ports[i].Subdomain[j] >= 'A' && ports[i].Subdomain[j] <= 'Z') || (ports[i].Subdomain[j] >= '0' && ports[i].Subdomain[j] <= '9')) {
				return fmt.Errorf("Subdomain is invalid")
			}
		}
	}
	old_ports, err := readServersPorts()
	if err != nil {
		return err
	}

	for i := 0; i < len(ports); i++ {
		for j := 0; j < len(old_ports); j++ {
			if ports[i].ServerPort == old_ports[j].ServerPort {
				return fmt.Errorf("port is already in use")
			}

			if ports[i].Subdomain == old_ports[j].Subdomain {
				return fmt.Errorf("Subdomain is already in use")
			}
		}
	}
	return nil
}

func add_ServerPortBin(ports []haproxyPorts) error {
	for i := 0; i < len(ports); i++ {
		_, err := db.Exec("INSERT INTO containerPorts (server_name, server_port, subdomain) VALUES (?, ?, ?)", ports[i].ServerName, ports[i].ServerPort, ports[i].Subdomain)
		if err != nil {
			return err
		}
	}
	return nil
}

func remove_ServerPortBin(port haproxyPorts) error {
	_, err := db.Exec("DELETE FROM containerPorts WHERE server_name = ?", port.ServerName)
	if err != nil {
		return err
	}
	return nil
}

func checkDockerData(port string) error {
	if port == "" {
		return fmt.Errorf("port is empty")
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return err
	}
	if portInt < 0 || portInt > 65535 {
		return fmt.Errorf("port is invalid")
	}
	return nil
}

// deixa estar com panic pq isto tem de ser mudado
func getDockerFiles(path string, rootPassword string, dockerfile string) {
	//open the file

	newFile, err := os.Create(filepath.Join(path, "Dockerfile"))
	if err != nil {
		panic(err)
	}
	defer newFile.Close()

	fileBytes := getFileBytes(dockerfile)

	fileBytes = bytes.ReplaceAll(fileBytes, []byte("passwordToChange"), []byte(rootPassword))
	//copy the contents
	_, err = newFile.Write(fileBytes)
	if err != nil {
		panic(err)
	}
}

func checkDockerImageExists(imageName string) bool {
	command := fmt.Sprintf("docker images -q %s", imageName)
	fmt.Println("Check if docker image exists: ", command)

	lines, err := RunCommandWithReturn(command)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return len(lines) >= 1
}

func checkDockerImageRunning(dockerName string) error {
	command := fmt.Sprintf("docker ps -a --filter name=%s", dockerName)
	fmt.Println("Check if docker image is running: ", command)

	lines, err := RunCommandWithReturn(command)
	if err != nil {
		return err
	}
	if len(lines) > 1 {
		return fmt.Errorf("docker image %s is already running", dockerName)
	}
	return nil
}

func buildDockerImage(path string, imageName string) error {
	existQ := checkDockerImageExists(imageName)
	if existQ {
		fmt.Println("Docker image already exists")
		return fmt.Errorf("docker image already exists")
	}

	command := fmt.Sprintf("docker build -t %s %s", imageName, path)
	fmt.Println(command)

	_, err := RunCommandWithReturn(command)

	if err != nil && !strings.Contains(err.Error(), "DONE") {
		fmt.Println(err)
		return err
	}

	existQ = checkDockerImageExists(imageName)

	if existQ {
		return nil
	}

	return fmt.Errorf("docker image %s was not found after being created", imageName)
}

func runDockerImage(imageName string, dockerName string, port string, open_ports []haproxyPorts) error {
	err := checkDockerData(port)
	if err != nil {
		fmt.Println("err1: ", err)
		return err
	}

	command := fmt.Sprintf("docker run -d -p %s:22 ", port)
	for i := 0; i < len(open_ports); i++ {
		command += fmt.Sprintf("-p %s:%s ", open_ports[i].ServerPort, open_ports[i].ServerPort)
	}
	command += fmt.Sprintf("--name %s %s", dockerName, imageName)

	fmt.Println(command)

	res, err := RunCommandWithReturn(command)
	if err != nil {
		fmt.Println("err2: ", err)
		return err
	}

	fmt.Println(res)

	err = checkDockerImageRunning(dockerName)
	if err != nil {
		return nil
	}

	return fmt.Errorf("docker image %s failed to run was not found after being created ", dockerName)
}

func setDockerPassword(password string, containerID string) error {
	command := fmt.Sprintf("docker exec %s bash -c 'echo -e \"%s\n%s\" | passwd root'", containerID, password, password)
	_, err := RunCommandWithReturn(command)
	if err != nil && !strings.Contains(err.Error(), "password updated successfully") {
		fmt.Println(err)
		return err
	}
	return nil
}

func getAllDockerInfoJson() []string {
	command := "docker ps -a --format '{{json .}}'"
	ret, err := RunCommandWithReturn(command)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return ret
}

func isPortInUse(port int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func generatePort() (string, error) {
	rand.Seed(uint64(time.Now().UnixNano())) // Ensure the seed is set to current time
	for i := 0; i < 10; i++ {
		port := 10000 + rand.Intn(55535)
		if !isPortInUse(port) {
			return strconv.Itoa(port), nil
		}
	}
	return "", errors.New("could not generate an available port after 10 attempts")
}

func runDocker(password string, dockerPort string, imageDockerName string, ServerName string, openPorts []haproxyPorts) error {

	containerExists := checkDockerImageExists(imageDockerName)

	if !containerExists {
		return fmt.Errorf("docker image %s does not exist", imageDockerName)
	}

	err := runDockerImage(imageDockerName, ServerName, dockerPort, openPorts)
	if err != nil {
		fmt.Println("runDocker: ", err)
		return err
	}

	add_ServerPortBin(openPorts)

	err = setDockerPassword(password, ServerName)
	if err != nil {
		fmt.Println("setDockerPassword: ", err)
		return err
	}

	return nil
}

func stopContainer(containerID string) error {
	command := fmt.Sprintf("docker stop %s", containerID)
	fmt.Println(command)
	res, err := RunCommandWithReturn(command)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(res)
	return nil
}

func deleteContainer(containerID string) error {
	command := fmt.Sprintf("docker rm %s", containerID)
	fmt.Println(command)
	res, err := RunCommandWithReturn(command)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(res)
	return nil
}

func startContainer(containerID string) error {
	command := fmt.Sprintf("docker start %s", containerID)
	fmt.Println(command)
	res, err := RunCommandWithReturn(command)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(res)
	restartHaProxy(HAPROXYPORT)
	return nil
}

func restartContainer(containerID string) error {
	command := fmt.Sprintf("docker restart %s", containerID)
	fmt.Println(command)
	res, err := RunCommandWithReturn(command)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(res)
	return nil
}

func findServerName(containerID string) string {
	command := fmt.Sprintf("docker inspect %s --format='{{.Name}}'", containerID)
	fmt.Println(command)
	res, err := RunCommandWithReturn(command)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	res2 := strings.ReplaceAll(res[0], "/", "")

	return res2
}

func checkIfDockerNameIsBuiltCommand(name string) bool {
	command := fmt.Sprintf("docker images -q %s", name)
	fmt.Println("Check if docker image exists: ", command)

	lines, err := RunCommandWithReturn(command)
	if err != nil {
		fmt.Println(err)
		return false
	}
	fmt.Println("lines: ", lines)
	return len(lines) >= 1
}

func checkIfDockerImageIsBuilt(name string, dockerFile string) bool {
	built := checkIfDockerNameIsBuiltCommand(name)

	if !built {
		path := createFolder(DOCKER_FILE_FOLDER_NAME)

		//place dockerfile on folder
		getDockerFiles(path, "temp123", dockerFile) // temporary way to get dockerFiles so need to change this

		err := buildDockerImage(path, name)
		if err != nil && !strings.Contains(err.Error(), "docker image already exists") {
			fmt.Println("runDocker: ", err)
			return false
		}
	}

	built2 := checkIfDockerNameIsBuiltCommand(name)

	return built2

}

func getDockerImagesJson() []string {
	command := "docker images --format '{{json .}}'"
	ret, err := RunCommandWithReturn(command)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return ret
}

func createDockerImage(imageName string, commands []string) error {
	startMainCommands := `
FROM ubuntu:20.04
RUN apt-get update && \
	apt-get install -y openssh-server sshfs curl && \
	apt-get clean
RUN mkdir /var/run/sshd
RUN echo 'root:passwordToChange' | chpasswd
RUN echo "Port 22" >> /etc/ssh/sshd_config && \
	echo "PermitRootLogin yes" >> /etc/ssh/sshd_config && \
	echo "PasswordAuthentication yes" >> /etc/ssh/sshd_config
`
	finalCommands := `
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]
`

	for i := 0; i < len(commands); i++ {
		startMainCommands += "RUN " + commands[i]
		if i != len(commands)-1 {
			startMainCommands += "\n"
		}
	}

	finalCommands = startMainCommands + finalCommands

	runCommand := fmt.Sprintf("docker build -t %s - <<EOF%sEOF", imageName, finalCommands)

	ret, err := RunCommandWithReturn(runCommand)
	if err != nil && !strings.Contains(err.Error(), "DONE") {
		fmt.Println(err)
		return err
	}

	fmt.Println(ret)

	_, err = db.Exec("INSERT INTO docker_images (name, commands) VALUES (?, ?)", imageName, strings.Join(commands, ","))
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func dockerInit() {

	//check if docker is installed
	_, err := RunCommandWithReturn("docker --version")
	if err != nil {
		fmt.Println("Docker is not installed")
		return
	}

	//check if docker is running
	_, err = RunCommandWithReturn("docker ps")
	if err != nil {
		fmt.Println("Docker is not running")
		return
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS containerPorts (server_name TEXT, server_port TEXT, subdomain TEXT)")
	if err != nil {
		return
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS docker_images (id INTEGER PRIMARY KEY, name TEXT, commands TEXT)")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Docker init")
}

func getDockerImagesNames() []string {
	rows, err := db.Query("SELECT * FROM docker_images")
	if err != nil {
		fmt.Println(err)
		return nil
	}

	var ret []string
	for rows.Next() {
		var id int
		var name string
		var commands string
		err = rows.Scan(&id, &name, &commands)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		//return json format with imageName and path, the path should be null
		ret = append(ret, name)
	}

	return ret
}

func removeDockerImageCommand(name string) error {
	command := fmt.Sprintf("docker rmi %s", name)
	fmt.Println(command)
	res, err := RunCommandWithReturn(command)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(res)
	return nil
}

func removeDockerImageSQL(name string) error {
	_, err := db.Exec("DELETE FROM docker_images WHERE name = ?", name)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

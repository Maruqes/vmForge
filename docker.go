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

	"golang.org/x/exp/rand"
)

const DOCKER_FILE_FOLDER_NAME = "/dockerFiles"

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
func getDockerFiles(path string, rootPassword string) {
	//open the file
	exampleData := "testData/DockerFileExample"

	newFile, err := os.Create(filepath.Join(path, "Dockerfile"))
	if err != nil {
		panic(err)
	}
	defer newFile.Close()

	fileBytes := getFileBytes(exampleData)

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
	fmt.Println("lines: ", lines)
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

func runDockerImage(imageName string, dockerName string, port string) error {
	err := checkDockerData(port)
	if err != nil {
		fmt.Println("err1: ", err)
		return err
	}

	command := fmt.Sprintf("docker run -d -p %s:22 --name %s %s", port, dockerName, imageName)
	fmt.Println(command)

	res, err := RunCommandWithReturn(command)
	if err != nil {
		fmt.Println("err2: ", err)
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
	fmt.Println(command)
	res, err := RunCommandWithReturn(command)
	if err != nil && !strings.Contains(err.Error(), "password updated successfully") {
		fmt.Println(err)
		return err
	}
	fmt.Println(res)
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

func runDocker(password string, dockerPort string, imageDockerName string, serverName string) error {

	containerExists := checkDockerImageExists(imageDockerName)

	if !containerExists {
		return fmt.Errorf("docker image %s does not exist", imageDockerName)
	}

	err := runDockerImage(imageDockerName, serverName, dockerPort)
	if err != nil {
		fmt.Println("runDocker: ", err)
		return err
	}

	err = setDockerPassword(password, serverName)
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

func checkIfDockerImageIsBuilt(name string) bool {
	built := checkIfDockerNameIsBuiltCommand(name)

	if !built {
		path := createFolder(DOCKER_FILE_FOLDER_NAME)

		//place dockerfile on folder
		getDockerFiles(path, "temp123") // temporary way to get dockerFiles so need to change this

		err := buildDockerImage(path, name)
		if err != nil {
			fmt.Println("runDocker: ", err)
			return false
		}
	}

	built2 := checkIfDockerNameIsBuiltCommand(name)

	return built2

}

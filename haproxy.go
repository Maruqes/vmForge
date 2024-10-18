package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type haproxy struct {
	serverName string
	serverInt  string
	serverPort string
}

// serverName, serverPort, subdomain
type haproxyPorts struct {
	serverName string
	serverPort string
	subdomain  string
}

const HAPROXY_FILE_FOLDER_NAME = "/haproxyFiles"
const HAPROXY_SERVERS_FILE_NAME = "/serversSave.bin"
const HAPROXY_SERVERS_PORTS_FILE_NAME = "/serversPortsSave.bin"

func checkServersProxyData(servers []haproxy) error {
	for i := 0; i < len(servers); i++ {
		if servers[i].serverName == "" {
			return fmt.Errorf("serverName is empty")
		}
		if servers[i].serverInt == "" {
			return fmt.Errorf("serverInt is empty")
		}
		if servers[i].serverPort == "" {
			return fmt.Errorf("serverPort is empty")
		}
	}

	for i := 0; i < len(servers); i++ {
		serverPortInt, err := strconv.Atoi(servers[i].serverPort)
		if err != nil {
			return err
		}
		if serverPortInt < 0 || serverPortInt > 65535 {
			return fmt.Errorf("serverPort is invalid")
		}
	}

	for i := 0; i < len(servers); i++ {
		for j := i + 1; j < len(servers); j++ {
			if servers[i].serverInt == servers[j].serverInt {
				return fmt.Errorf("serverInt is not unique")
			}

			if servers[i].serverPort == servers[j].serverPort {
				return fmt.Errorf("serverPort is not unique")
			}

			if servers[i].serverName == servers[j].serverName {
				return fmt.Errorf("serverName is not unique")
			}
		}
	}
	return nil
}

func checkCertificate() error {
	currentDir, _ := os.Getwd()
	// cd into haproxyFiles
	err := os.Chdir(filepath.Join(currentDir, HAPROXY_FILE_FOLDER_NAME))
	if err != nil {
		return err
	}

	if _, err := os.Stat(currentDir + HAPROXY_FILE_FOLDER_NAME + "/ssh.pem"); os.IsNotExist(err) {
		command := "openssl req -x509 -newkey rsa:4096 -nodes -sha256 -subj '/CN=localhost' -keyout private.pem -out cert.pem"
		RunCommand(command)

		command = "awk '1' cert.pem private.pem > ssh.pem"
		RunCommand(command)
	}

	// cd back to the original directory
	err = os.Chdir(currentDir)
	if err != nil {
		return err
	}
	return nil

}

func killOtherHaProxyProcesses() {
	command := "pkill haproxy"
	RunCommand(command)
}

var cmdHaProxy *exec.Cmd

func runHaProxuProcess() {
	killOtherHaProxyProcesses()

	command := "haproxy -D -f " + "." + HAPROXY_FILE_FOLDER_NAME + "/haproxy.cfg -p ." + HAPROXY_FILE_FOLDER_NAME + "/haproxy.pid"
	log.Println(command)
	log.Println("HAProxy is running")

	cmdHaProxy = exec.Command("bash", "-c", command)
	go RunCommandWithCMD(command, cmdHaProxy)
}

func checkSaveServersExistance() error {
	currentDir, _ := os.Getwd()

	if _, err := os.Stat(currentDir + HAPROXY_FILE_FOLDER_NAME + HAPROXY_SERVERS_FILE_NAME); os.IsNotExist(err) {
		_, err := os.Create(currentDir + HAPROXY_FILE_FOLDER_NAME + HAPROXY_SERVERS_FILE_NAME)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat(currentDir + HAPROXY_FILE_FOLDER_NAME + HAPROXY_SERVERS_PORTS_FILE_NAME); os.IsNotExist(err) {
		_, err := os.Create(currentDir + HAPROXY_FILE_FOLDER_NAME + HAPROXY_SERVERS_PORTS_FILE_NAME)
		if err != nil {
			return err
		}
	}
	return nil
}

func readServersBinary() []haproxy {
	currentDir, _ := os.Getwd()

	checkSaveServersExistance()

	serversBin, err := os.ReadFile(currentDir + HAPROXY_FILE_FOLDER_NAME + HAPROXY_SERVERS_FILE_NAME)

	if err != nil {
		panic(err)
	}

	var servers []haproxy

	serversStr := string(serversBin)
	if len(serversStr) == 0 {
		return servers
	}
	serversStr = serversStr[:len(serversStr)-1]
	serversStrArr := strings.Split(serversStr, "\n")

	for i := 0; i < len(serversStrArr); i++ {
		server := strings.Split(serversStrArr[i], " ")
		servers = append(servers, haproxy{server[0], server[1], server[2]})
	}

	return servers
}

func createNewHAPROXYServer(serverName string, serverInt string, serverPort string) error {
	oldServers := readServersBinary()
	oldServers = append(oldServers, haproxy{serverName, serverInt, serverPort})

	err := checkServersProxyData(oldServers)
	if err != nil {
		return err
	}

	currentDir, _ := os.Getwd()

	servers := readServersBinary()
	servers = append(servers, haproxy{serverName, serverInt, serverPort})

	err = checkSaveServersExistance()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(currentDir+HAPROXY_FILE_FOLDER_NAME+HAPROXY_SERVERS_FILE_NAME, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for i := 0; i < len(servers); i++ {
		_, err = file.WriteString(servers[i].serverName + " " + servers[i].serverInt + " " + servers[i].serverPort + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func get_containerIP(serverName string) string {
	command := "docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' " + serverName

	res, err := RunCommandWithReturn(command)
	if err != nil {
		return ""
	}
	return res[0]
}

func createHaProxyFiles(path string, mainPort string, servers []haproxy) error {
	err := checkServersProxyData(servers)

	if err != nil {
		return err
	}

	//open the file
	headExampleData := "testData/sshCFG_head_Example"
	backendExampleData := "testData/sshCFG_backend_Example"

	portFrontend := "testData/PortFrontendExample"
	portBackend := "testData/PortBackendExample"

	newFile, err := os.Create(filepath.Join(path, "haproxy.cfg"))
	if err != nil {
		return err
	}
	defer newFile.Close()

	headFileBytes := getFileBytes(headExampleData)
	backendFileBytes := getFileBytes(backendExampleData)

	headFileBytes = bytes.ReplaceAll(headFileBytes, []byte("mainPort"), []byte(mainPort))

	_, err = newFile.Write(append(headFileBytes, '\n'))
	if err != nil {
		return err
	}

	haproxyPortsExample := []haproxyPorts{{"vmForge_123", "901", "test1"}, {"vmForge_123", "902", "test2"}}

	//needs serverName, serverPort, \, serverip, servermainport, backendName
	portFrontendFileBytes := getFileBytes(portFrontend)
	portBackendFileBytes := getFileBytes(portBackend)

	for i := 0; i < len(haproxyPortsExample); i++ {
		tempPortFrontend := make([]byte, len(portFrontendFileBytes))
		copy(tempPortFrontend, portFrontendFileBytes)
		tempPortFrontend = bytes.ReplaceAll(tempPortFrontend, []byte("host_serverName"), []byte(haproxyPortsExample[i].serverName+"_"+haproxyPortsExample[i].serverPort))
		tempPortFrontend = bytes.ReplaceAll(tempPortFrontend, []byte("subdomain"), []byte(haproxyPortsExample[i].subdomain))
		tempPortFrontend = bytes.ReplaceAll(tempPortFrontend, []byte("serverIP"), []byte(SERVER_LINK))
		tempPortFrontend = bytes.ReplaceAll(tempPortFrontend, []byte("http_backendName"), []byte(haproxyPortsExample[i].serverName+"_"+haproxyPortsExample[i].subdomain))

		_, err = newFile.Write(append(tempPortFrontend, '\n', '\n'))
		if err != nil {
			return err
		}
	}

	for i := 0; i < len(haproxyPortsExample); i++ {
		sv_ip := get_containerIP(haproxyPortsExample[i].serverName)
		if sv_ip == "" {
			fmt.Errorf("server not found trying to set port")
			continue
		}
		tempPortBackend := make([]byte, len(portBackendFileBytes))
		copy(tempPortBackend, portBackendFileBytes)
		tempPortBackend = bytes.ReplaceAll(tempPortBackend, []byte("ContainerIP"), []byte(sv_ip))
		tempPortBackend = bytes.ReplaceAll(tempPortBackend, []byte("serverContPort"), []byte(haproxyPortsExample[i].serverPort))
		tempPortBackend = bytes.ReplaceAll(tempPortBackend, []byte("http_backendName"), []byte(haproxyPortsExample[i].serverName+"_"+haproxyPortsExample[i].subdomain))

		_, err = newFile.Write(append(tempPortBackend, '\n', '\n'))
		if err != nil {
			return err
		}
	}

	for i := 0; i < len(servers); i++ {
		tempbackend := make([]byte, len(backendFileBytes))
		copy(tempbackend, backendFileBytes)
		tempbackend = bytes.ReplaceAll(tempbackend, []byte("serverName"), []byte(servers[i].serverName))
		tempbackend = bytes.ReplaceAll(tempbackend, []byte("serverInt"), []byte(servers[i].serverInt))
		tempbackend = bytes.ReplaceAll(tempbackend, []byte("serverPort"), []byte(servers[i].serverPort))

		_, err = newFile.Write(append(tempbackend, '\n', '\n'))
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteHAPROXYServer(serverName string) error {
	currentDir, _ := os.Getwd()

	servers := readServersBinary()

	checkSaveServersExistance()

	file, err := os.OpenFile(currentDir+HAPROXY_FILE_FOLDER_NAME+HAPROXY_SERVERS_FILE_NAME, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for i := 0; i < len(servers); i++ {
		if servers[i].serverName == serverName {
			servers = append(servers[:i], servers[i+1:]...)
			break
		}
	}

	file.Truncate(0)

	for i := 0; i < len(servers); i++ {
		_, err = file.WriteString(servers[i].serverName + " " + servers[i].serverInt + " " + servers[i].serverPort + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func runHaProxy(mainHaProxyPort string) error {
	path := createFolder(HAPROXY_FILE_FOLDER_NAME)
	err := checkCertificate()
	if err != nil {
		return err
	}

	servers := readServersBinary()
	fmt.Println("servers: ", servers)

	err = createHaProxyFiles(path, mainHaProxyPort, servers)
	if err != nil {
		return err
	}
	runHaProxuProcess()
	return nil
}

func restartHaProxy(mainHaProxyPort string) {
	fmt.Println("Restarting HAProxy")
	servers := readServersBinary()
	fmt.Println("servers: ", servers)

	err := createHaProxyFiles("."+HAPROXY_FILE_FOLDER_NAME, mainHaProxyPort, servers)
	if err != nil {
		fmt.Println(err)
	}

	go func() {
		command := "haproxy -D -f " + "." + HAPROXY_FILE_FOLDER_NAME + "/haproxy.cfg -p ." + HAPROXY_FILE_FOLDER_NAME + "/haproxy.pid -sf $(cat ." + HAPROXY_FILE_FOLDER_NAME + "/haproxy.pid)"
		fmt.Println(command)
		res, err := RunCommandWithReturn(command)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(res)
	}()
}

func openPort(serverName string, serverPort string) error {
	return nil
}

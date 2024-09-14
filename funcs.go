package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// RunCommand runs a command in the background and outputs its stdout/stderr to the console
func RunCommand(command string) {

	// Create the bash command
	cmd := exec.Command("bash", "-c", command)

	// Get stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error creating stdout pipe:", err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println("Error creating stderr pipe:", err)
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting command:", err)
		return
	}

	var outputWg sync.WaitGroup
	outputWg.Add(2)

	go func() {
		defer outputWg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	go func() {
		defer outputWg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	outputWg.Wait()

}

func RunCommandWithReturn(command string) ([]string, error) {
	// Create the bash command
	cmd := exec.Command("bash", "-c", command)

	// Get stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error creating stdout pipe:", err)
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println("Error creating stderr pipe:", err)
		return nil, err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting command:", err)
		return nil, err
	}

	var outputWg sync.WaitGroup
	outputWg.Add(2)
	var output []string
	var outputErr []string

	go func() {
		defer outputWg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			output = append(output, line)
		}
	}()

	go func() {
		defer outputWg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			outputErr = append(output, line)
		}
	}()

	outputWg.Wait()

	if len(outputErr) > 0 {
		return nil, fmt.Errorf("error running command: %v", outputErr)
	}
	return output, nil
}

func RunCommandWithCMD(command string, cmd *exec.Cmd) {

	// Get stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error creating stdout pipe:", err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println("Error creating stderr pipe:", err)
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting command:", err)
		return
	}

	var outputWg sync.WaitGroup
	outputWg.Add(2)

	go func() {
		defer outputWg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	go func() {
		defer outputWg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	outputWg.Wait()
}

// TerminateCommand terminates the currently running command
func TerminateCommand(cmd *exec.Cmd) {

	if cmd != nil {
		if err := cmd.Process.Kill(); err != nil {
			fmt.Println("Error killing command:", err)
		} else {
			fmt.Println("Command terminated")
		}
		cmd = nil
	}
}

func createFolder(name string) string {
	currentDir, _ := os.Getwd()
	newDir := filepath.Join(currentDir, name)

	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		//create the folder
		os.Mkdir(newDir, 0755)
	}

	return newDir
}

func getFileBytes(path string) []byte {
	//open the file
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	//get the file size
	fileInfo, _ := file.Stat()
	fileSize := fileInfo.Size()

	//read the file
	data := make([]byte, fileSize)
	_, err = file.Read(data)
	if err != nil {
		panic(err)
	}

	return data
}

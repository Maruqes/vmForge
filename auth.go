package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/rand"
)

const AUTH_TOKEN_LENGHT = 64

var db *sql.DB

type Auth struct {
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func (auth *Auth) createWebToken() string {
	rand.Seed(uint64(time.Now().UnixNano()))

	b := make([]rune, AUTH_TOKEN_LENGHT)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// MAP FOR USER / TOKENS
type UserLoggedIn struct {
	username string
	token    string
}

var usersLoggedIn = make(map[string]UserLoggedIn)

func (auth *Auth) loginWithWebToken(username string, token string) (bool, error) {
	if token == "" {
		return false, fmt.Errorf("empty token")
	}

	user, ok := usersLoggedIn[username]
	if !ok {
		return false, fmt.Errorf("user not logged in")
	}

	if user.token == token {
		return true, nil
	}

	return false, nil
}

func (auth *Auth) refreshWebToken(username string, old_token string) (string, error) {
	user, ok := usersLoggedIn[username]
	if !ok {
		return "", fmt.Errorf("user not logged in")
	}

	if user.token != old_token {
		return "", fmt.Errorf("invalid token")
	}

	token := auth.createWebToken()
	usersLoggedIn[username] = UserLoggedIn{username: username, token: token}
	return token, nil
}

func (auth *Auth) checkUserPassCombination(username string, password string) (bool, error) {
	if username == "" || password == "" {
		return false, fmt.Errorf("empty fields")
	}

	var storedHash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&storedHash)
	if err != nil {
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		return false, nil
	}

	return true, nil
}

func hashPassword(password string) (string, error) {

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (auth *Auth) login(username string, password string) (string, error) {
	if username == "" || password == "" {
		return "", fmt.Errorf("empty fields")
	}

	loggedIn, err := auth.checkUserPassCombination(username, password)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("wrong username or password")
		}
		return "", err
	}

	token := ""
	if loggedIn {
		token = auth.createWebToken()
		usersLoggedIn[username] = UserLoggedIn{username: username, token: token}
	} else {
		return "", fmt.Errorf("wrong username or password")
	}

	if token == "" {
		return "", fmt.Errorf("error creating token")
	}

	fmt.Println("User " + username + " logged in")
	return token, nil
}

func checkUserData(username string, password string) error {
	if username == "" || password == "" {
		return fmt.Errorf("empty fields")
	}

	if !isValidInput(username) || !isValidInput(password) {
		return fmt.Errorf("fields contain wierd characters")
	}

	return nil
}

// todo: detect used username
func doesUserExist(username string) (bool, error) {
	var storedUsername string
	err := db.QueryRow("SELECT username FROM users WHERE username = ?", username).Scan(&storedUsername)
	if err != nil {
		return false, nil
	}

	return true, nil
}

func (auth *Auth) register(username string, password string) error {
	if username == "" || password == "" {
		return fmt.Errorf("empty fields")
	}

	err := checkUserData(username, password)
	if err != nil {
		return err
	}

	userExist, err := doesUserExist(username)
	if err != nil {
		return err
	}

	if userExist {
		return fmt.Errorf("user already exists")
	}

	hashedPassword, err := hashPassword(password)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, hashedPassword)
	if err != nil {
		return err
	}

	fmt.Println("User " + username + " registered")
	return nil
}

func createSqlFile() error {
	path, err := os.Getwd()
	if err != nil {
		return err
	}

	dbFilePath := filepath.Join(path, "db.sql")
	if _, err := os.Stat(dbFilePath); os.IsNotExist(err) {
		newFile, err := os.Create(dbFilePath)
		if err != nil {
			return err
		}
		defer newFile.Close()
	}

	db, err = sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, username TEXT, password_hash TEXT)")
	if err != nil {
		return err
	}

	return nil
}

func (auth *Auth) init() error {
	fmt.Println("Auth init")
	err := createSqlFile()

	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func (auth *Auth) printAllLoggedUsers() {
	for _, user := range usersLoggedIn {
		fmt.Println(user.username)
	}
}
func (auth *Auth) getAdminsArrName() []string {
	rows, err := db.Query("SELECT username FROM users")
	if err != nil {
		fmt.Println(err)
		return nil
	}

	var ret []string
	for rows.Next() {
		var username string
		err = rows.Scan(&username)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		ret = append(ret, username)
	}

	return ret
}

func removeLoggedInUser(username string) {
	delete(usersLoggedIn, username)
}

func (auth *Auth) removeUser(username string) error {
	_, err := db.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		return err
	}

	removeLoggedInUser(username)
	return nil
}

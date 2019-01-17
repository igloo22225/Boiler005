package main

import (
	"bufio"
	"encoding/base32"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
	"github.com/skratchdot/open-golang/open"
)

var debugMode = false

func introText() {
	fmt.Println("********** PLEASE READ BELOW BEFORE CONTINUING **********") //Give the user an overview and the obligatory sec warning.
	fmt.Println("Boiler005 - A BoilerKey that poses no immediate risk in any direct sense.")
	fmt.Println("Boiler005 will take a Duo activation URL and will generate a QR code for use with other 2-factor authentication apps.")
	fmt.Println("Boiler005 will store the token as an image on the disk. You will be prompted to delete it at the end.")
	fmt.Println("Now then -")
	fmt.Println("Browse to the BoilerKey settings page (https://purdue.edu/boilerkey) and select \"Set up a new Duo Mobile BoilerKey\"")
	fmt.Println("Follow any instructions (except for installing the Duo app), naming the device what you will.")
	fmt.Println("When you are presented with a QR code and URL (Step 4 of 6), copy and paste the URL below.")
	fmt.Println()
	fmt.Print("URL: ")
}

func getActivationCode(url string) string {
	location := strings.Index(url, "activate/") //Look through the URL to find where the token should be, pull it and return it
	location = location + 9
	activationToken := string(url[location : location+20])
	return activationToken
}

func getHOTPToken(body []byte) string {
	bodyString := string(body) //Not the cleanest way of dealing with this issue, though I don't need any of the other data and it works just as well.
	location := strings.Index(bodyString, "\"hotp_secret\":")
	location = location + 16
	HOTPToken := string(bodyString[location : location+32])
	return HOTPToken
}

func registerAsClient(activationToken string) string {
	client := &http.Client{}
	data := url.Values{}
	data.Set("app_id", "com.duosecurity.duomobile.app.DMApplication") //This data has to be sent as post data as Duo doesn't want to give it to just any device
	data.Set("app_version", "2.3.3")
	data.Set("app_build_number", "323206")
	data.Set("full_disk_encryption", "false")
	data.Set("manufacturer", "Google")
	data.Set("model", "Pixel") //Ah yes, Golang Pixel!
	data.Set("platform", "Android")
	data.Set("jailbroken", "False")
	data.Set("version", "6.0")
	data.Set("language", "EN")
	data.Set("customer_protocol", "1")
	fullURL := "https://api-1b9bef70.duosecurity.com/push/v2/activation/" + activationToken
	if debugMode == true {
		fmt.Println("DUO Request URL: " + fullURL)
	}
	req, _ := http.NewRequest("POST", fullURL, strings.NewReader(data.Encode())) //Emulate the app and request the token
	req.Header.Set("User-Agent", "okhttp/3.11.0")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if debugMode == true {
		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)
		fmt.Println("response Body:", string(body))
	}
	HOTPToken := getHOTPToken(body)
	return HOTPToken
}

func validateURL(urlIn string) bool {
	_, err := url.ParseRequestURI(urlIn)
	if err != nil {
		return false
	}
	return true
}

func getDuoData() string {
	url := "" //Take the user's URL
	reader := bufio.NewReader(os.Stdin)
	url, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading URL, please restart.") //If for whatever reason the string isn't readable, yell
		os.Exit(1)
	}
	if validateURL(url) == false {
		fmt.Println("URL looks wrong! Please restart.")
		os.Exit(1)
	}
	activationToken := getActivationCode(url)
	if debugMode == true {
		fmt.Println("Your activation token appears to be " + activationToken + ".")
	}
	fmt.Println("Attempting to register with Duo...")
	HOTPToken := registerAsClient(activationToken)
	return HOTPToken
}

func generateQRCode(HOTPToken string) {
	fmt.Println("Generating QR Code...")
	QRCode := base32.StdEncoding.EncodeToString([]byte(HOTPToken))
	QRCode = "otpauth://hotp/Purdue_Boiler_Key?secret=" + QRCode
	err := qrcode.WriteFile(QRCode, qrcode.Medium, 256, "bk.png")
	if err != nil {
		panic(err)
	}
	fmt.Println("Opening QR code using your default photo viewer...")
	open.Run("bk.png")
}

func cleanup() {
	fmt.Println("Please scan the QR code with your 2-factor application.")
	fmt.Println("Once you have done so, you should delete the image from local storage (in order to protect your account).")
	fmt.Println("Before continuing, please close the local program displaying your image.")
	fmt.Println("To do so, simply hit enter. If you would rather keep the image (a *really* bad idea unless you protect it), enter \"n\", and press enter.")
	selection := "" //Take the user's URL
	reader := bufio.NewReader(os.Stdin)
	selection, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading selection, please restart.") //If for whatever reason the string isn't readable, yell
		return
	}
	selection = strings.ToLower(strings.Trim(selection, " \r\n"))
	if selection == "n" {
		fmt.Println("Image has not been removed! Please remember to secure it!")
	} else {
		os.Remove("bk.png")
		fmt.Println("Image has been removed.")
	}
	fmt.Println("Program complete.")
}

func main() {
	//Intro text
	introText()
	//Take input, go get json, find the HOTP token
	HOTPToken := getDuoData()
	//Generate a QR code with the token
	generateQRCode(HOTPToken)
	//Wait for the user to confirm cleanup
	cleanup()

}

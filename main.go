package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Juniper/go-netconf/netconf"
	xj "github.com/basgys/goxml2json"
	"golang.org/x/crypto/ssh"
)

const prefix = "veryuniqueattrprefix-"

func main() {
	http.HandleFunc("/editmodule", editModule)
	http.HandleFunc("/getmodule", getModule)
	http.HandleFunc("/addmodule", addModule)

	log.Print(http.ListenAndServe(":8080", nil))
}

func addModule(w http.ResponseWriter, req *http.Request) {
	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	req.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := req.FormFile("myFile")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	defer file.Close()
	fmt.Printf("Uploaded File: %+v\n", handler.Filename)

	dst, err := os.Create(handler.Filename)
	defer dst.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy the uploaded file to the created file on the filesystem
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	time.Sleep(time.Second)
	command := "sysrepoctl -i " + handler.Filename
	//command := "ls -lrth"
	out, err := exec.Command("bash", "-c", command).Output()

	if err != nil {
		fmt.Printf("Error: %s", err)
	} else {
		fmt.Println("Module stored successfully")
	}
	output := string(out[:])
	fmt.Println(output)
}

func getModule(w http.ResponseWriter, req *http.Request) {

	session := startSSHSession()
	if session == nil {
		return
	}
	defer session.Close()

	xml := `<led-data xlmns="urn:sysrepo:my"></led-data>`
	r, err := session.Exec(netconf.MethodGet("subtree", xml))
	if err != nil {
		fmt.Println("err:1", err)
		return
	}

	// fmt.Println("r: ", r)
	// fmt.Println("\n\n\n\n")
	// fmt.Println("r.rawreply ", r.RawReply)

	data := ConvertToJSON(r.RawReply) // // s = "<xmlns>Hi</ns>" --> []byte --> "{\"foo\":{\"baz\": [1,2,3]}}"( 12 342 42 12 )

	JSONdata := make(map[string]interface{})
	// json.Unmarshal([]byte(s), &JSONdata)
	json.Unmarshal(data, &JSONdata)
	// fmt.Println(JSONdata)
	// yash, _ := json.Marshal(JSONdata)
	// fmt.Println(string(yash))
	w.Header().Set("Content-Type", "application/json")
	// url := "localhost:8080"

	// fmt.Fprint(w, string(yash))
	json.NewEncoder(w).Encode(JSONdata)
	//

	// fmt.Println(">>>>>", r.RawReply)
	// j := strings.Split(fmt.Sprint(r), "#203")
	// fmt.Println(">>>>>>", strings.Split(j[1], ""))
	// ConvertToJSON(fmt.Sprint(r))
	// r, err = session.Exec(netconf.MethodGetConfig("running"))
	// if err != nil {
	// 	fmt.Println("err:2", err)
	// 	return
	// }

	// fmt.Println("\n\n Running Confing \n")
	// displayReply(r.RawReply)

}
func editModule(w http.ResponseWriter, req *http.Request) {
	session := startSSHSession()
	if session == nil {
		return
	}
	defer session.Close()
	// // Define the new YANG module content
	//newModule := `<led-data xmlns="urn:sysrepo:my"><turned-on>17</turned-on></led-data>`
	//newModule := ""
	//xml.NewDecoder(req.Body).Decode(&newModule)
	data, _ := ioutil.ReadAll(req.Body)
	fmt.Println("@@@@@@", string(data))
	// // Connect to the Netconf server
	// conn, err := netconf.DialSSH("localhost:830", netconf.SSHConfigPassword("vvdn", os.Getenv("syspwd")))
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// fmt.Println(session.ServerCapabilities)
	// fmt.Println(session.SessionID)

	// Add the new YANG module to the Netconf server
	//_, err = conn.EditConfig(netconf.ConfigData(newModule), "merge", netconf.ConfigTargetRunning)
	// 	data := `<led-data xmlns="urn:sysrepo:my">
	//     <turned-on>17</turned-on>
	// </led-data>`
	_, err := session.Exec(netconf.MethodEditConfig("running", string(data)))
	if err != nil {
		fmt.Println("err:0", err)
		return
	}
	// fmt.Println(r.RawReply)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, "New YANG module edited successfully")
	//ConvertToJSON(fmt.Sprint(r))
}

func startSSHSession() *netconf.Session {
	sshConfig := &ssh.ClientConfig{
		User:            "netconf",
		Auth:            []ssh.AuthMethod{ssh.Password("netconf")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		//Timeout:         time.Duration(*timeout) * time.Second,
	}
	session, err := netconf.DialSSH("localhost:830", sshConfig)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	return session
}

func prettyPrint(b string) string {
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(b), "", "  "); err == nil {
		return string(out.Bytes())
	}
	return ""
}

func displayReply(rawReply string) {
	xml := strings.NewReader(rawReply)
	json, err := xj.Convert(xml)
	if err != nil {
		log.Fatal("Something went sore ... XML is invalid!")
	}
	fmt.Println(prettyPrint(json.String()))

}

func ConvertToJSON(xmlstring string) []byte {

	xml := strings.NewReader(xmlstring)
	// Decode XML document

	json, err := xj.Convert(xml)
	if err != nil {
		panic("That's embarrassing...")
		//return nil
	}

	//	fmt.Println("#######", json.String())
	return json.Bytes() // "  10 39 431"
}

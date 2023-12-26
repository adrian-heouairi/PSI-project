package main

import (
	"crypto/sha256"
	//"fmt"
	//"log"
	"net"
	"net/http"
	"strings"
    "time"
    "sync"
    "strconv"
)

var jchConn *net.UDPConn

func createDownloadDirAndCd() {
	_, err := os.Stat(DOWNLOAD_DIR)
	if os.IsNotExist(err) {
		err = os.Mkdir(DOWNLOAD_DIR, 0755)
		checkErr(err)
	}

    bodyAsByteSlice, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LOGGING_FUNC(err)
	}

    return resp, bodyAsByteSlice
}

func getPeers() []string{
    _, bodyAsByteSlice := httpGet(SERVER_ADDRESS + PEERS_PATH)
    listOfPeersAsString := string(bodyAsByteSlice)
    listOfPeers := strings.Split(listOfPeersAsString,"\n")
    printNumberedList(listOfPeers)
    return listOfPeers
}

func getAdressesOfPeer(peerName string) []string {
    resp, bodyAsByteSlice := httpGet(SERVER_ADDRESS + PEERS_PATH + "/" + peerName + "/addresses")

	if resp.StatusCode == 404 {
		LOGGING_FUNC(peerName + " is not known by server")
		return make([]string, 0)
	}
	
    return strings.Split(string(bodyAsByteSlice), "\n")
}

func getRootOfPeer(peerName string) []byte {
    resp, bodyAsByteSlice := httpGet(SERVER_ADDRESS+PEERS_PATH+"/"+peerName+"/root")

	if resp.StatusCode == 204 {
		LOGGING_FUNC(peerName + " has not declared a root yet!")
		return make([]byte, 0)
	} else if resp.StatusCode == 404 {
		LOGGING_FUNC(peerName + "is not known by server!")
		return make([]byte, 0)
	}
	
    return bodyAsByteSlice
}

func createHello() udpMsg {
    var helloMsg udpMsg
    helloMsg.Id = rand.Uint32()
    helloMsg.Type = 2
    extensions := make([]byte, 4)
    name := OUR_PEER_NAME
    nameAsBytes := []byte(name)
    var res = append(extensions, nameAsBytes...)
    helloMsg.Body = res
    helloMsg.Length = uint16(len(res))
    return helloMsg
}

func keepConnectionAlive() {
    serverUdpAddresses := getAdressesOfPeer(SERVER_PEER_NAME)

	// Server address
	serverAddr, err := net.ResolveUDPAddr("udp", serverUdpAddresses[0])
	checkErr(err)

	// Establish a connection
	jchConn, err = net.DialUDP("udp", nil, serverAddr)
	checkErr(err)
	defer jchConn.Close()

	buffer := make([]byte, 1048576)
    
    helloMsg := createHello()
for {

	listAllFilesOfPeer("jch.irif.fr")

	/*download_and_save_file := func(file_name string, hash []byte, content []byte) {}
	download_and_save_file = func(file_name string, hash []byte, content []byte) {
		f, err := os.OpenFile(file_name, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Failed to create")
			log.Fatal(err)
		}
		defer f.Close()
		datumMsg := sendAndReceiveMsg(createMsg(GET_DATUM, hash))
		fmt.Println(udpMsgToString(datumMsg))
		if datumMsg.Type != DATUM {
			LOGGING_FUNC("Not a datum msg")
			//return
		}
		hashToCheck := datumMsg.Body[:32]
		if datumMsg.Body[DATUM_TYPE_INDEX] == CHUNK {
			if check_data_integrity(hashToCheck, datumMsg.Body[32:datumMsg.Length]) {
				content = append(content, datumMsg.Body[33:datumMsg.Length]...)
			} else {
				LOGGING_FUNC("CORRUPTED CHUNK")
			}
		} else if datumMsg.Body[DATUM_TYPE_INDEX] == TREE {
			// bigFile/Tree : 1 + (32 *[2:32])
			nbElt := (datumMsg.Length - 33) / uint16(HASH_SIZE)
			for i := 0; i < int(nbElt); i++ {
				download_and_save_file(file_name, datumMsg.Body[33+i*32:65+i*32], content)
			}
			fmt.Println("CONTENT before write : " + string(content))
			bytesWritten, err := f.Write(content)
			if err != nil {
				LOGGING_FUNC("WRINTING GOES WRONG")
			}
			fmt.Println("wrote : " + fmt.Sprint(bytesWritten) + "bytes")

		} else if datumMsg.Body[DATUM_TYPE_INDEX] == DIRECTORY {
			hashes := getDataHashes(datumMsg)
			var fk string
			i := 0
			for key := range hashes {
				fk = key
				if i == 4 {
					break
				}
				i++
			}
			download_and_save_file("bidule", hashes[fk], content)
		}
	}

	rootJchREST := getRootOfPeer("jch.irif.fr")

	rootDatumReply := sendAndReceiveMsg(createMsg(GET_DATUM, rootJuliuszUDP))
	// hash puis contenu
	if !check_data_integrity(
		rootDatumReply.Body[:32],
		rootDatumReply.Body[32:rootDatumReply.Length]) {
		LOGGING_FUNC("DATUM IS CORRUPTED")
	}
<<<<<<< HEAD

    publicKeyMsg := byteSliceToUdpMsg(buffer)

    if publicKeyMsg.Type != PUBLIC_KEY {
        LOGGING_FUNC("Invalid PublicKey message")
    }

    publicKeyReplyMsg := udpMsg{publicKeyMsg.Id, PUBLIC_KEY_REPLY, 0, make([]byte, 0)}

    _, err = conn.Write(udpMsgToByteSlice(publicKeyReplyMsg))
    if err != nil {
        LOGGING_FUNC(err)
    }

    _, _, err = conn.ReadFromUDP(buffer)
    if err != nil {
		LOGGING_FUNC(err)
	}

    rootMsg := byteSliceToUdpMsg(buffer)

    if rootMsg.Type != ROOT {
        LOGGING_FUNC("Invalid Root message")
    }

    hasher := sha256.New()
    rootReplyMsg := udpMsg{rootMsg.Id, ROOT_REPLY, 32, hasher.Sum(nil)}

    _, err = conn.Write(udpMsgToByteSlice(rootReplyMsg))
    if err != nil {
        LOGGING_FUNC(err)
 }
     time.Sleep(30 * time.Second) 
     fmt.Println("After waiting 30 seconds")
 }
}
func printNumberedList(list []string) {
    for i:= 0 ; i < len(list) - 1; i++{
        fmt.Println(strconv.Itoa(i + 1) + " - " + list[i])
    }
=======
	hashes := getDataHashes(rootDatumReply)
	fmt.Println(hashes)
	download_and_save_file("images", rootDatumReply.Body[:32], []byte{})*/
>>>>>>> 3f298843e7df187b0f67fbe7ff84e7159c458fd2
}
func UI() {
    fmt.Println("PEER CLIENT")
    fmt.Println("1 - Get peers list")
    fmt.Println("2 - Get addresses of a peer")
    fmt.Println("3 - Get root of a peer")
    var i int

    fmt.Print("Type a number[1..3]: ")
    fmt.Scan(&i)
    switch i {
    case 1:
        fmt.Println("Here is the list of peers :")
        getPeers()
    case 2:
        listOfPeers := getPeers()
        fmt.Println("Which pair are you interesseted by[1.." + strconv.Itoa(len(listOfPeers)) + "] :")
        fmt.Scan(&i)
        fmt.Println(listOfPeers[i-1] + " addresses are : ")
        printNumberedList(getAdressesOfPeer(listOfPeers[i-1]))
    }
    fmt.Println("Your number is:", i)
}

func main() {
UI()
    /*
    PAY ATTENTION THIS CODE COMES FROM CHATGPT AND NEED TO BE REFACTORED
    */
var wg sync.WaitGroup

	// Start the goroutine and increment the WaitGroup counter
	wg.Add(1)
	go func() {
		defer wg.Done()
		keepConnectionAlive()
	}()

	// Your other main function logic...

	// Wait for the goroutine to finish before exiting
	wg.Wait()
	fmt.Println("Program exited")
go keepConnectionAlive()
/*
END OF CHATGPT CODE
*/
}

download_and_save_file := func(file_name string, hash []byte, content []byte) {}
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
hashes := getDataHashes(rootDatumReply)
fmt.Println(hashes)
download_and_save_file("images", rootDatumReply.Body[:32], []byte{})

// Send and receive root
mr := sendAndReceiveMsg(createMsg(ROOT, hasher.Sum(nil)))
fmt.Println(udpMsgToString(mr))
rootJuliuszUDP := mr.Body

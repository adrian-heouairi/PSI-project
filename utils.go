package main

import (
	"container/list"
	"os"
	"sync"
)

func mkdir(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.Mkdir(path, 0755)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

func writeChunk(path string, chunk []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY | os.O_CREATE | os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(chunk)
	if err != nil {
		return err
	}

	return nil
}

func threadSafeAppendToList(list *list.List, mutex *sync.RWMutex, elem any) {
	mutex.Lock()
	defer mutex.Unlock()

	list.PushBack(elem)
}

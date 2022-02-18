package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"google.golang.org/protobuf/proto"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"protobuf_demo_server/my_message"
)

type User struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Message struct {
	Id      int    `json:"id"`
	Content string `json:"content"`
	Sender  User   `json:"sender"`
}

var logger = log.Default()

var tempFile *os.File = nil

func getTempFile() (*os.File, error) {
	var err error
	if tempFile == nil {
		tempFile, err = ioutil.TempFile(".", "socket-")
		if err != nil {
			return nil, fmt.Errorf("error getting tempFile: %v\n", err)
		}
	}
	return tempFile, nil
}

func writeToDisk(message *Message) error {
	file, err := getTempFile()
	if err != nil {
		return fmt.Errorf("error getting temp file: %v\n", err)
	}
	err = file.Truncate(0)
	if err != nil {
		return err
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}
	msgAsProtobuf := my_message.Message{
		Id:      int64(message.Id),
		Content: message.Content,
		Sender:  &my_message.User{
			Id:   int64(message.Sender.Id),
			Name: message.Sender.Name,
		},
	}
	buf, err := proto.Marshal(&msgAsProtobuf)
	if err != nil {
		return fmt.Errorf("error marshaling: %v\n", err)
	}
	_, err = file.Write(buf)
	if err != nil {
		return fmt.Errorf("error writing: %v\n", err)
	}
	return nil
}

func startSocketListener(address string) {
	listener, err := net.Listen("unix", address)
	if err != nil {
		logger.Fatalf("error dialing socket: %v\n", err)
	}
	logger.Printf("Listening on socket %v\n", address)
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Printf("error reading accepted connection: %v\n", err)
			continue
		}
		content, err := io.ReadAll(conn)
		if err != nil {
			logger.Printf("error reading: %v\n", err)
			continue
		}
		decoder := json.NewDecoder(bytes.NewReader(content))
		var msg Message
		err = decoder.Decode(&msg)
		if err != nil {
			logger.Printf("error decoding: %v\n", err)
			continue
		}
		err = writeToDisk(&msg)
		if err != nil {
			logger.Printf("error writing to disk: %v\n", err)
			continue
		}
		fmt.Println(string(content))
	}
}

func main() {
	defer func() {
		_ = tempFile.Close()
	}()
	// use an abstract namespace socket to avoid having to manage a file
	listenIn := flag.String("listenTo", "sock1", "the name of the abstract namespace socket to listen on")
	flag.Parse()
	listenTo := fmt.Sprintf("@%s", *listenIn)
	startSocketListener(listenTo)
}

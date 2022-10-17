package plugin

import (
	"bufio"
	"compress/gzip"
	"github.com/vearne/grpcreplay/buffpool"
	"github.com/vearne/grpcreplay/protocol"
	"github.com/vearne/gtimer"
	slog "github.com/vearne/simplelog"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type FileDirInput struct {
	codec     protocol.Codec
	msgChan   chan *protocol.Message
	timer     *gtimer.SuperTimer
	path      string
	readDepth int
	// smallest timestamp
	benchmarkTimestamp int64
	reader             *ReinforcedReader
}

func NewFileDirInput(codec string, path string, readDepth int) *FileDirInput {
	var in FileDirInput
	in.codec = protocol.GetCodec(codec)
	in.msgChan = make(chan *protocol.Message, 100)
	in.timer = gtimer.NewSuperTimer(3)
	in.path = path
	in.readDepth = readDepth
	in.benchmarkTimestamp = 0

	in.init()
	return &in
}

func (in *FileDirInput) init() {
	// scan directory
	files, err := getFilesAndDirs(in.path)
	if err != nil {
		slog.Fatal("FileDirInput-scan directory:%v", err)
	}
	in.reader = NewReinforcedReader(files, in.codec)
	msgList := make([]*protocol.Message, 0, in.readDepth)

	slog.Debug("readDepth:%v", in.readDepth)
	for i := 0; i < in.readDepth; i++ {
		msg, err := in.reader.ReadMessage()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				slog.Fatal("ReinforcedReader read:%v", err)
			}
		}
		msgList = append(msgList, msg)
		if i == 0 {
			in.benchmarkTimestamp = msg.Meta.Timestamp
		} else if msg.Meta.Timestamp < in.benchmarkTimestamp {
			in.benchmarkTimestamp = msg.Meta.Timestamp
		}
	}
	slog.Info("benchmarkTimestamp:%v, len(msgList):%v",
		in.benchmarkTimestamp, len(msgList))
	for i := 0; i < len(msgList); i++ {
		msg := msgList[i]
		addTaskToTimer(in, msg)
	}
}

func addTaskToTimer(in *FileDirInput, msg *protocol.Message) {
	d := time.Duration(msg.Meta.Timestamp - in.benchmarkTimestamp)
	slog.Debug("delay:%v", time.Now().Add(d))
	task := gtimer.NewDelayedItemFunc(
		time.Now().Add(d),
		msg,
		func(t time.Time, param interface{}) {
			message := param.(*protocol.Message)
			in.msgChan <- message
			// Keep the total number of messages in the priority queue constant
			newMessage, err := in.reader.ReadMessage()
			if err != nil {
				if err == io.EOF {
					slog.Debug("All files are read")
				} else {
					slog.Error("ReinforcedReader read:%v", err)
				}
				return
			} else {
				addTaskToTimer(in, newMessage)
			}
		},
	)
	in.timer.Add(task)
}

func (in *FileDirInput) Read() (msg *protocol.Message, err error) {
	msg = <-in.msgChan
	return msg, nil
}

func (in *FileDirInput) Close() error {
	return in.reader.Close()
}

type ReinforcedReader struct {
	sync.Mutex
	codec     protocol.Codec
	file      *os.File
	reader    *bufio.Reader
	filepaths []string
	index     int
	EOF       bool
}

func NewReinforcedReader(filepaths []string, codec protocol.Codec) *ReinforcedReader {
	var r ReinforcedReader
	r.index = 0
	r.codec = codec

	sort.Strings(filepaths)
	r.filepaths = filepaths

	slog.Debug("create ReinforcedReader, files:%v", filepaths)
	if len(r.filepaths) <= 0 {
		slog.Fatal("ReinforcedReader:no file to read")
	}

	var err error
	r.file, r.reader, err = createReader(r.filepaths[0])
	if err != nil {
		slog.Fatal("read file [%v]:%v", r.filepaths[0], err)
	}
	return &r
}

func createReader(path string) (file *os.File, reader *bufio.Reader, err error) {
	var gz *gzip.Reader
	// gzip file
	if strings.HasSuffix(path, ".gz") {
		file, err = os.Open(path)
		if err != nil {
			return
		}
		gz, err = gzip.NewReader(file)
		reader = bufio.NewReader(gz)
		if err != nil {
			return
		}
	} else {
		file, err = os.Open(path)
		if err != nil {
			return
		}
		return file, bufio.NewReader(file), nil
	}
	return
}

func (r *ReinforcedReader) Close() error {
	return r.file.Close()
}

func (r *ReinforcedReader) NextFile() error {
	if r.index+1 < len(r.filepaths) {
		var err error
		// close old file
		err = r.file.Close()
		if err != nil {
			return err
		}
		// normal circumstances, try next file
		r.index++
		slog.Info("switch to file:%v", r.filepaths[r.index])
		r.file, r.reader, err = createReader(r.filepaths[r.index])
		if err != nil {
			return err
		}
		return nil
	}
	r.EOF = true
	slog.Info("All files are read")
	return io.EOF
}

func (r *ReinforcedReader) ReadMessage() (*protocol.Message, error) {
	r.Lock()
	defer r.Unlock()

	if r.EOF {
		return nil, io.EOF
	}

	slog.Debug("[start]ReinforcedReader.ReadMessage()")
	var line []byte
	var err error

	bf := buffpool.GetBuff()
	defer buffpool.PutBuff(bf)

	first := true
	for first || len(line) > 1 {
		// line contains delimiter
		line, err = r.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				// switch to next file
				err = r.NextFile()
				if err == nil {
					// 完成了切换
					first = true
					continue
				} else {
					slog.Debug("[end]ReinforcedReader.ReadMessage()")
					return nil, err
				}
			}
		}

		first = false
		if len(line) > 1 {
			bf.Write(line)
		}
	}

	data := bf.Bytes()
	slog.Debug("error:%v, file:%v, filepaths:%v, len(data):%v, content:%v",
		err, r.filepaths[r.index], r.filepaths, len(data), string(data))
	data = data[0 : len(data)-1]

	var msg protocol.Message

	err = r.codec.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}

	slog.Debug("[end]ReinforcedReader.ReadMessage()")
	return &msg, nil
}

/*
-rw-r--r--  1 root  wheel   299464 10 14 13:50 capture-2022-10-14T05-50-39.473.log.gz
-rw-r--r--  1 root  wheel   299153 10 14 13:50 capture-2022-10-14T05-50-41.733.log.gz
-rw-r--r--  1 root  wheel   300325 10 14 13:50 capture-2022-10-14T05-50-44.328.log.gz
-rw-r--r--  1 root  wheel  7333254 10 14 13:50 capture.log
*/
func getFilesAndDirs(dirPth string) (files []string, err error) {
	fileInfoList, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, err
	}

	PthSep := string(os.PathSeparator)
	files = make([]string, 0)
	for _, fi := range fileInfoList {
		if fi.IsDir() { // 目录, 递归遍历
			continue
		}
		name := fi.Name()
		if !strings.HasPrefix(name, "capture") {
			continue
		}

		if strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".log.gz") {
			files = append(files, dirPth+PthSep+fi.Name())
		}
	}
	return files, nil
}

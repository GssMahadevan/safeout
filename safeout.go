package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"

	// "bufio"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"syscall"
	"time"

	// "golang.org/x/sys/unix"
	"github.com/goccy/go-yaml"
)

const (
	DEF_PERMS = 0644
)

var (
	cfg     = flag.String("cfg", "./safeout.yaml", "Json file for all safestdout fifo files & related config")
	cpu     = flag.Int("cpu", getSafeCpuCount(), "Max cpus to be used for writing/reading logs")
	watch   = flag.Bool("watch", false, "Watch configuration file for changes") //TODO
	toutMs  = flag.Int("toutS", 500, "Read timeout for channel read in shutdown")
	verbose = flag.Bool("verbose", false, "Verbose log messages") //TODO

	outperms = flag.String("outperms", "0644", "Outfile's permissions  for monitoring stdin for log rotate without yaml configuration")
	outfile  = flag.String("outfile", "", `Outfile for monitoring stdin for log rotate without yaml configuration\n
	But each program needs to run extra instance of 'safeout' program
	`)

	outMax  = flag.Int("outmax", 10_000_000, "Maximum outfile size")
	bufsize = flag.Int("bufsize", 8192, "Buf size for writing to regular file")
	// perms_re = regexp.MustCompile("^0[0-7]{3}$|^0o[0-7]{3}$")
	perms_re = regexp.MustCompile("^0[0-7]{3}$")
	cfg_info []Cfg
	config   Config

	done   = make(chan bool, 1)
	stopC  []*stopChannel //chan bool
	active = true
)

type stopChannel struct {
	name     string
	isClosed bool
	ch       chan bool
}
type Cfg0 struct {
	FifoName  string `json:"name"`     // Fifo file  path on which  stdout/stderr are routed by client processes and this go process reads
	FileName  string `json:"filename`  // file where this go process writes logs with size check
	MaxSize   int    `json:"maxsize"`  // maximum file size after which new file is started
	MaxFiles  int    `json:"maxfiles"` // maximum file for backup
	Compress  bool   `json:"compress"` // compress backup file
	PermsFifo string `json:"permFifo"` // file permissions in octal format 0[0-7]{3}
	PermsFile string `json:"permFile"` // file permissions in octal format 0[0-7]{3}
}

type Com struct {
	MaxFiles int `yaml:"maxfiles"`
	MaxSize  int `yaml:"maxsize"`
}

type Perms struct {
	PermsFifo string `yaml:"permFifo"`
	PermsFile string `yaml:"permFile"`
}
type Cfg struct {
	Com      `yaml:"com"`
	Perms    `yaml:"perms`
	FifoName string `yaml:"fifoName"`
	FileName string `yaml:"fileName"`
	Compress bool   `yaml:"compress"`
	Parent   string `yaml:",omitempty"`
}

type Config struct {
	Description string           `yaml:"description"`
	Version     string           `yaml:"version"`
	Perms       map[string]Perms `yaml:"perms"`
	Coms        map[string]Com   `yaml:"common"`
	Cfgs        map[string]Cfg   `yaml:"safeouts"`
}

func (me *Cfg) S() string {
	return fmt.Sprintf("fifo:%s, file:%s, maxSize:%d", me.FifoName, me.FileName, me.MaxSize)
}

func init() {
	flag.Parse()
	// log.SetFlags(log.Llongfile | log.LstdFlags)
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	runtime.GOMAXPROCS(*cpu)
}

func newStopChannel(name string) *stopChannel {
	r := &stopChannel{name: name, isClosed: false}
	r.ch = make(chan bool, 1)
	return r
}
func getSafeCpuCount() (n int) {
	n = runtime.NumCPU()
	if n > 4 {
		n = 2
	} else {
		n = 1
	}
	return n
}

func Start() int {
	return safeStdOutStart()
}

func WaitForSignals() {
	setSignalHandler(done)
	<-done
	if *verbose {
		log.Printf("Signalling Shutting down ...")
	}
	active = false
	safeStdOutStop()
	time.Sleep(5 * time.Second)
	if *verbose {
		log.Printf(" finished shutdown\n")
	}
}

func safeStdOutStart() int {
	// loadJsonCfg()
	loadYamlCfg()
	siz := len(cfg_info)
	cnt := 0
	// if siz == 0 {
	// 	log.Printf("No config file entries to monitor\n")
	// 	// return 0
	// }
	for i := 0; i < siz; i++ {
		sc := newStopChannel(cfg_info[i].S())
		stopC = append(stopC, sc)
	}

	for i, cfg := range cfg_info {
		if !ensureFifo(&cfg) {
			continue
		}
		cnt++
		go handleCfg(cfg, stopC[i])
	}
	if "" != *outfile { // handle outfile in case simple safeout execution like old unix pipe stuff
		sc := newStopChannel(*outfile)
		stopC = append(stopC, sc)
		cnt++
		go handleStdin(*outfile, *outperms, sc)

	}
	return cnt
}

func loadYamlCfg() {
	yml, err := ioutil.ReadFile(*cfg)
	if err != nil {
		if *verbose {
			log.Printf("can't loadYamlCfg '%s',  err:%v\n", *cfg, err)
		}
	}
	err = yaml.Unmarshal(yml, &config)
	if err != nil {
		if *verbose {
			log.Printf("loadYamlCfg  can't Unmarshal yaml '%s',  err:%v\n", *cfg, err)
		}
	}

	for k, v_ := range config.Cfgs {
		v := v_
		v.Parent = k
		log.Printf("Loading config for %s , val:%+v\n", k, v)
		log.Printf("cfg:%p, Com:%p, Perms:%p\n", &v, &v.Com, &v.Perms)
		cfg_info = append(cfg_info, v)
	}
	// log.Printf("\n\ncfg_info is %+v\n\n", cfg_info)
}
func setSignalHandler(done chan bool) {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-sigs
		if *verbose {
			log.Printf("Got signal %v\n", sig)
		}
		done <- true
	}()
}
func safeStdOutStop() {
	for i := 0; i < len(stopC); i++ {
		go sendSafeShutMsg(i, stopC[i])
	}
}

func safeCloseChan(stopChan *stopChannel) {
	recoverFromPanic(fmt.Sprintf("safeCloseChan_%s", stopChan.name))
	if *verbose {
		log.Printf("safeCloseChan for:%s, %p", stopChan.name, stopChan.ch)
	}
	if !stopChan.isClosed {
		close(stopChan.ch)
	}

}
func sendSafeShutMsg(i int, stopChan *stopChannel) {
	recoverFromPanic(fmt.Sprintf("sendSafeShutMsg_%s", stopChan.name))
	if *verbose {
		log.Printf("sendSafeShutMsg stopChan:%s, chan:%p", stopChan.name, stopChan.ch)
	}
	select {
	case stopChan.ch <- true:
	case <-time.After(time.Duration(*toutMs) * time.Millisecond):
		log.Printf("sendSafeShutMsg got timeout %d sec", *toutMs)
	}
}

func recoverFromPanic(ctx string) {
	if r := recover(); r != nil {
		log.Printf("Recovered from panic ctx:%s, r:%v\n", ctx, r)
	}
}

func getSize(cfg *Cfg) int64 {
	return getSize0((cfg.FileName))
}
func getSize0(fileName string) int64 {
	finfo, err := os.Stat(fileName)
	if err != nil {
		//log.Printf("can't do state")
		return -1
	}
	return finfo.Size()
}
func isClosedChan(c <-chan bool) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}
func handleCfg(cfg_ Cfg, stopChan *stopChannel) {
	var cfg *Cfg = &cfg_
	defer recoverFromPanic(fmt.Sprintf("handleCfg %v", cfg))
	defer safeCloseChan(stopChan)

	outF := getWritableFile(cfg, true, true)
	if outF == nil {
		log.Printf("handleCfg can't get out file :%s", cfg.S())
		return // TODO what to do when shutdown stopC message comes for reading?
	}
	defer outF.Close()

	fifo := getFifo(cfg)
	if fifo == nil {
		log.Printf("handleCfg can't get fifo file :%s", cfg.S())
		return // TODO what to do when shutdown stopC message comes for reading?
	}
	defer fifo.Close()

	buf := make([]byte, *bufsize)
	fifoEof := false
	n := 0
	var err error
	var fSize int64 = getSize(cfg)

	if fSize < 0 {
		log.Printf("can't stat the file:%s", cfg.S())
		return
	}

	lp := 0
OUT:
	for {
		if !active {
			break
		}
		lp++

		fifo = ensureFifoOpen(&fifoEof, fifo, cfg)
		if fifo == nil {
			break OUT
		}
		select {
		case <-stopChan.ch:
			log.Printf("handleCfg got shutdown msg:%s", stopChan.name)
			break OUT
		default:
			n, err = fifo.Read(buf)
		}
		if err == io.EOF {
			fifoEof = true
		}
		if os.IsTimeout(err) {
			continue
		}
		if n > 0 {

			if !writen(outF, buf, n, cfg.FileName) {
				break OUT
			}

			fSize += int64(n)
			if fSize > int64(cfg.MaxSize) {
				outF.Close()
				renameFile(cfg)
				outF = getWritableFile(cfg, false, true)
				if outF == nil {
					log.Printf("can't reopen file for writing %s\n", cfg.S())
					break OUT
				}
				fSize = 0
			}

		}
	}
	stopChan.isClosed = true
	if *verbose {
		log.Printf("handleCfg exiting stopChan  %s, ptr:%p", stopChan.name, stopChan.ch)
	}
}

func handleStdin(fileName, filePerm string, stopChan *stopChannel) {
	defer recoverFromPanic("handleStdin")
	defer safeCloseChan(stopChan)

	outF := getWritableFile0(fileName, filePerm, true, true)
	if outF == nil {
		log.Printf("handleCfg can't get out file :%s", fileName)
		return // TODO what to do when shutdown stopC message comes for reading?
	}
	defer outF.Close()

	stdin := os.Stdin
	if stdin == nil {
		log.Printf("handleCfg can't get file :%s")
		return // TODO what to do when shutdown stopC message comes for reading?
	}
	defer stdin.Close()

	buf := make([]byte, *bufsize)

	n := 0
	var err error
	var fSize int64 = getSize0(fileName)

	if fSize < 0 {
		log.Printf("can't stat the file:%s", fileName)
		return
	}

	lp := 0
OUT:
	for {
		if !active {
			break
		}
		lp++

		// stdin = ensureFifoOpen(&fifoEof, stdin, cfg)
		// if stdin == nil {
		// 	break OUT
		// }
		select {
		case <-stopChan.ch:
			log.Printf("handleCfg got shutdown msg:%s", stopChan.name)
			break OUT
		default:
			n, err = stdin.Read(buf)
		}
		if err == io.EOF {
			break
		}

		if os.IsTimeout(err) {
			continue
		}
		if n > 0 {

			if !writen(outF, buf, n, fileName) {
				break OUT
			}

			fSize += int64(n)
			if fSize > int64(*outMax) {
				outF.Close()
				renameFile0(fileName)
				outF = getWritableFile0(fileName, filePerm, false, true)
				if outF == nil {
					log.Printf("can't reopen file for writing %s\n", fileName)
					break OUT
				}
				fSize = 0
			}

		}
	}
	stopChan.isClosed = true
	if *verbose {
		log.Printf("handleCfg exiting stopChan  %s, ptr:%p", stopChan.name, stopChan.ch)
	}
}

func writen(outF *os.File, buf []byte, n int, fileName string) bool {
	off := 0
	nleft := n
	for nleft > 0 {
		bp := buf[off:n]
		//log.Printf("nleft:%8d, off:%8d, n:%8d, lenB:%8d, cap:%8d\n", nleft, off, n, len(bp), cap(bp))
		nW, err := outF.Write(bp)
		if err == io.ErrShortWrite {
			if nW > 0 {
				off += nW
				nleft -= nW
			}
			continue
		} else if err != nil {
			log.Printf("Can't write to file :%s, err:%v, nw:%d\n", fileName, err, nW)
			return false
		}
		off += nW
		nleft -= nW
	}
	return true
}
func renameFile(cfg *Cfg) bool {
	return renameFile0(cfg.FileName)
}
func renameFile0(fn string) bool {
	backupName := fmt.Sprintf("%s.backup", fn)
	err := os.Rename(fn, backupName)
	if err != nil {
		log.Printf("can't rename file %s, err:%v\n", fn, err)
		// runScript() //TODO removed this script
		return false
	}
	return true
}
func runScript() {
	cmdGoVer := &exec.Cmd{
		Path:   "/opt/bin/check1.sh",
		Args:   []string{},
		Stdout: os.Stdout,
		Stderr: os.Stdout,
	}
	if err := cmdGoVer.Run(); err != nil {
		fmt.Println("runScript Error:", err)
	}

}
func ensureFifoOpen(isEof *bool, f *os.File, cfg *Cfg) *os.File {
	if !*isEof {
		return f
	}
	if f != nil {
		f.Close()
	}
	f = getFifo(cfg)
	if f != nil {
		*isEof = false
	}
	return f
}
func getPerm(perm, file string) (ret uint32) {
	if !perms_re.Match([]byte(perm)) {
		log.Printf("Warning Bad permission format:%s file:%s\n", perm, file)
		return DEF_PERMS
	}
	v, err := strconv.ParseInt(perm, 8, 64)
	if err != nil {
		log.Printf("Warning Bad permission value:%s, file:%s\n", perm, file)
		return DEF_PERMS
	}
	return uint32(v)
}
func ensureFifo(cfg *Cfg) bool {
	if finfo, err := os.Stat(cfg.FifoName); err == nil {
		mode := finfo.Mode()
		if (mode & os.ModeNamedPipe) != 0 { // file fifo type
			return true
		}
	} else if os.IsNotExist(err) {
		err := syscall.Mkfifo(cfg.FifoName, getPerm(cfg.PermsFifo, cfg.FifoName))
		if err != nil {
			log.Printf("ensureFifo can't create %+v , err:%v\n", cfg, err)
			return false
		}
		return true
	}
	return false
}
func getFifo(cfg *Cfg) (f *os.File) {
	// f, err := os.Open(cfg.FifoName)
	// f, err := os.OpenFile(cfg.FifoName, os.O_RDONLY|syscall.O_NONBLOCK, 0) // O_RDWR
	f, err := os.OpenFile(cfg.FifoName, os.O_RDONLY, 0)
	if err != nil {
		log.Printf("getFifo Can't get fifo for reading  for:%v, err:%s\n", cfg.S(), err)
		return nil
	}
	return f
}
func getWritableFile(cfg *Cfg, isAppend, isCreate bool) (f *os.File) {
	return getWritableFile0(cfg.FileName, cfg.PermsFile, isAppend, isCreate)
}
func getWritableFile0(fileName, permsFile string, isAppend, isCreate bool) (f *os.File) {
	p := os.FileMode(getPerm(permsFile, fileName))
	mode := os.O_WRONLY
	if isAppend {
		mode |= os.O_APPEND
	}
	if isCreate {
		mode |= os.O_CREATE
	}
	f, err := os.OpenFile(fileName, mode, p)
	if err != nil {
		log.Printf("Can't get writable file for:%+v, mode:%v, err:%v\n", fileName, mode, err)
		return nil
	}
	return f
}
func loadJsonCfg() {
	jsonFile, err := os.Open(*cfg)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Fatalf("loadJsonCfg Open err:%v\n", err)
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf("loadJsonCfg ReadAll err:%v\n", err)
	}
	// var cfgs []Cfg

	json.Unmarshal([]byte(byteValue), &cfg_info)
	// fmt.Printf("%+v\n", logs)
	for i := 0; i < len(cfg_info); i++ {
		c := cfg_info[i]
		log.Printf("%03d. cfg:%+v\n", i, c)
	}
}

func main() {
	n := Start()
	if n > 0 {
		WaitForSignals()
	}
}

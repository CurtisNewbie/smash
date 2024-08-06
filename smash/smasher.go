package smash

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/curtisnewbie/miso/miso"
)

var (
	instructions SmashInstructions = SmashInstructions{}
	customClient *http.Client
)

func init() {
	// maximize the number of connections possible, disable redirect
	customClient = &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 2000
	t.MaxIdleConnsPerHost = 2000
	t.IdleConnTimeout = time.Second * 60
	customClient.Transport = t
}

func PrepareInstructions(rail miso.Rail) (SmashInstructions, error) {
	if v, ok := CliSmashInstruction(); ok {
		instructions.Add(v)
	} else {
		file, err := InstructionFilePath(rail)
		if err != nil {
			return SmashInstructions{}, err
		}
		err = LoadInstructionFile(rail, file)
		if err != nil {
			return SmashInstructions{}, err
		}
		instructions.AddAll(ConfSmashInstructions())
	}
	return instructions, nil
}

func singleSmash(rail miso.Rail, ins Instruction) {
	rail.Debugf("Preparing request to %v %v", ins.Method, ins.Url)
	cli := miso.NewTClient(rail, ins.Url).
		UseClient(customClient).
		AddHeaders(ins.Headers)

	var r *miso.TResponse
	switch ins.Method {
	case http.MethodGet:
		r = cli.Get()
	case http.MethodPut:
		r = cli.Put(bytes.NewReader([]byte(ins.Payload)))
	case http.MethodPost:
		r = cli.Post(bytes.NewReader([]byte(ins.Payload)))
	case http.MethodDelete:
		r = cli.Delete()
	case http.MethodHead:
		r = cli.Head()
	case http.MethodOptions:
		r = cli.Options()
	}

	if r.Err != nil {
		rail.Errorf("Endpoint %v %v returns error, %v", ins.Method, ins.Url, r.Err)
		return
	}

	rct := r.Resp.Header.Get("Content-Type")
	var rs string = "...binary..."
	if !strings.Contains(strings.ToLower(rct), "octet-stream") {
		s, err := r.Str()
		if err != nil {
			rail.Errorf("Endpoint %v %v returns error, %v", ins.Method, ins.Url, r.Err)
			return
		} else {
			rs = s
		}
	}
	if miso.IsDebugLevel() {
		rail.Debugf("Endpoint %v %v returns %v, %v, %+v", ins.Method, ins.Url, r.StatusCode, rs, r.Resp.Header)
	} else {
		rail.Infof("Endpoint %v %v returns %v", ins.Method, ins.Url, r.StatusCode)
	}
}

func doSmash(rail miso.Rail, exitWhenDone bool, instructions ...Instruction) {
	var instWg sync.WaitGroup // waitGroup for instructions

	for j := range instructions {
		inst := instructions[j]

		if inst.Parallelism < 1 {
			inst.Parallelism = 1
		}
		instWg.Add(1)

		go func(rail miso.Rail, inst Instruction) {
			defer instWg.Done()

			var totalTime int64
			var paraWg sync.WaitGroup // waitGroup for parallel requests

			for i := 0; i < inst.Parallelism; i++ {
				paraWg.Add(1)
				go func() {
					defer paraWg.Done()
					istart := time.Now()
					singleSmash(rail, inst)
					atomic.AddInt64(&totalTime, int64(time.Since(istart)))
				}()
			}
			paraWg.Wait()

			rail.Infof("\n\n\n>>> Instruction finished, '%v %v', took %v, on average: %v each, total parallel requests: %v <<< \n\n",
				inst.Method, inst.Url, time.Duration(totalTime), time.Duration(totalTime/int64(inst.Parallelism)), inst.Parallelism)
		}(rail.NextSpan(), inst)
	}

	instWg.Wait()

	if exitWhenDone {
		if !miso.HasScheduledJobs() { // we only have runOnce tasks
			os.Exit(0) // force the server to exit, not the best way of doing it, but this app has nothing serious either
		}
	}
}

// schedule instructions that are executed periodically until interrupted
func scheduleSmashing(rail miso.Rail, si SmashInstructions) error {
	cronInst := si.CronInstructions()
	for i := range cronInst {
		inst := cronInst[i]
		miso.ScheduleCron(miso.Job{
			Name:            fmt.Sprintf("inst-%v", i),
			Cron:            inst.Cron,
			CronWithSeconds: true,
			Run: func(rail miso.Rail) error {
				doSmash(rail, false, inst)
				return nil
			},
		})
	}
	return nil
}

func smashImmediately(rail miso.Rail, si SmashInstructions) error {
	doSmash(rail, true, si.RunOnceInstructions()...)
	return nil
}

var (
	cliDebug = flag.Bool("debug", false, "Debug")
)

func StartSmashing() error {
	miso.SetProp(miso.PropServerEnabled, false)
	miso.SetProp(miso.PropAppName, "smash")
	miso.SetProp(miso.PropProdMode, true)

	flag.Parse()
	if *cliDebug {
		miso.SetLogLevel("debug")
	}

	miso.PreServerBootstrap(func(rail miso.Rail) error {
		instr, err := PrepareInstructions(rail)
		if err != nil {
			return fmt.Errorf("failed to prepare instructions, %v", err)
		}
		instructions = instr
		return scheduleSmashing(rail, instructions)
	})

	miso.PostServerBootstrapped(func(rail miso.Rail) error {
		return smashImmediately(rail, instructions)
	})

	miso.BootstrapServer(os.Args)
	return nil
}

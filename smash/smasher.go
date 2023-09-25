package smash

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/curtisnewbie/gocommon/client"
	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/server"
)

var (
	instructions SmashInstructions
	customClient *http.Client
)

func init() {
	// maximize the number of connections possible
	customClient = &http.Client{Timeout: 10 * time.Second}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 2000
	t.MaxIdleConnsPerHost = 2000
	t.IdleConnTimeout = time.Second * 60
	customClient.Transport = t
}

func PrepareInstructions(rail common.Rail) (SmashInstructions, error) {
	file, err := InstructionFilePath(rail)
	if err != nil {
		return SmashInstructions{}, err
	}
	err = LoadInstructionFile(rail, file)
	if err != nil {
		return SmashInstructions{}, err
	}

	instructions := PackSmashInstructions()
	return instructions, nil
}

func singleSmash(rail common.Rail, ins Instruction) {
	rail.Debugf("Preparing request to %v %v", ins.Method, ins.Url)
	cli := client.NewTClient(rail, ins.Url, customClient).
		AddHeaders(ins.Headers)

	var r *client.TResponse
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

	s, err := r.ReadStr()
	if err != nil {
		rail.Errorf("Endpoint %v %v returns error, %v", ins.Method, ins.Url, r.Err)
		return
	}
	rail.Debugf("Endpoint %v %v returns %v, %v", ins.Method, ins.Url, r.StatusCode, s)
}

func doSmash(rail common.Rail, exitWhenDone bool, instructions ...Instruction) {
	var instWg sync.WaitGroup // waitGroup for instructions

	for j := range instructions {
		inst := instructions[j]

		if inst.Parallelism < 1 {
			inst.Parallelism = 1
		}
		instWg.Add(1)

		go func(rail common.Rail, inst Instruction) {
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
		if !common.HasScheduler() { // we only have runOnce tasks
			os.Exit(0) // force the server to exit, not the best way of doing it, but this app has nothing serious either
		}
	}
}

// schedule instructions that are executed periodically until interrupted
func scheduleSmashing(rail common.Rail, si SmashInstructions) error {
	cronInst := si.CronInstructions()
	for i := range cronInst {
		inst := cronInst[i]
		common.ScheduleCron(inst.Cron, true, func() {
			doSmash(rail, false, inst)
		})
	}
	return nil
}

func smashImmediately(rail common.Rail, si SmashInstructions) error {
	doSmash(rail, true, si.RunOnceInstructions()...)
	return nil
}

func StartSmashing() error {
	common.SetProp(common.PROP_SERVER_ENABLED, false)
	common.SetProp(common.PROP_APP_NAME, "smash")
	common.SetProp(common.PROP_PRODUCTION_MODE, true)
	common.SetProp(common.PROP_LOGGING_LEVEL, "info")

	server.PreServerBootstrap(func(rail common.Rail) error {
		instr, err := PrepareInstructions(rail)
		if err != nil {
			return fmt.Errorf("failed to prepare instructions, %v", err)
		}
		instructions = instr

		return scheduleSmashing(rail, instructions)
	})

	server.PostServerBootstrapped(func(rail common.Rail) error {
		return smashImmediately(rail, instructions)
	})

	server.BootstrapServer(os.Args)
	return nil
}

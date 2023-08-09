package smash

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/curtisnewbie/gocommon/client"
	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/server"
)

var (
	instructions SmashInstructions
)

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
	rail.Infof("Preparing request to %v %v", ins.Method, ins.Url)
	cli := client.NewDefaultTClient(rail, ins.Url).
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
	defer r.Close()

	s, err := r.ReadStr()
	if err != nil {
		rail.Errorf("Endpoint %v %v returns error, %v", ins.Method, ins.Url, r.Err)
		return
	}
	rail.Infof("Endpoint %v %v returns: %v", ins.Method, ins.Url, s)
}

func doSmash(rail common.Rail, exitWhenDone bool, instructions ...Instruction) {
	var wg sync.WaitGroup

	for j := range instructions {
		inst := instructions[j]

		parall := inst.Parallism
		if parall < 1 {
			parall = 1
		}

		for i := 0; i < parall; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()
				singleSmash(rail, inst)
			}()
		}
	}

	wg.Wait()

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

	server.PreServerBootstrap(func(rail common.Rail) error {
		instr, err := PrepareInstructions(rail)
		if err != nil {
			return fmt.Errorf("failed to prepare instructions, %v", err)
		}
		rail.Debugf("Instructions: %v", instructions)
		instructions = instr

		return scheduleSmashing(rail, instructions)
	})

	server.PostServerBootstrapped(func(rail common.Rail) error {
		return smashImmediately(rail, instructions)
	})

	server.BootstrapServer(os.Args)
	return nil
}

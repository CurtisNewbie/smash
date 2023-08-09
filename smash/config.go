package smash

import (
	"fmt"
	"os"

	"github.com/curtisnewbie/gocommon/common"
)

const (
	INSTRUCTION_EXAMPLE = `
instructions:
  - url: "http://localhost:8080/ping" # instruction with cron that runs periodically
    method: PUT
    parallism: 100
    cron: "*/1 * * * * ?"
    headers:
      - "Content-Type": "application/json"
    payload: '{ "purpose": "get wrecked my boi!!!" }'

  - url: "http://localhost:8080/pong" # instruction without cron that only run once
    method: GET
    parallism: 100
    headers:
      - "Content-Type": "application/json"
`
)

type Instruction struct {
	Cron      string
	Parallism int
	Url       string
	Method    string
	Headers   map[string]string
	Payload   string
}

type SmashInstructions struct {
	Instructions []Instruction `mapstructure:"instructions"`
}

func (si SmashInstructions) filter(predicate common.Predicate[Instruction]) []Instruction {
	filtered := []Instruction{}
	for i := range si.Instructions {
		inst := si.Instructions[i]
		if predicate(inst) {
			filtered = append(filtered, inst)
		}
	}
	return filtered
}

func (si SmashInstructions) RunOnceInstructions() []Instruction {
	return si.filter(func(t Instruction) bool {
		return common.IsBlankStr(t.Cron)
	})
}

func (si SmashInstructions) CronInstructions() []Instruction {
	return si.filter(func(t Instruction) bool {
		return !common.IsBlankStr(t.Cron)
	})
}

func InstructionFilePath(rail common.Rail) (string, error) {
	file := common.GetPropStr(PROP_INSTRUCTION_PATH)
	if common.IsBlankStr(file) {
		return "", fmt.Errorf("please specifiy file path using '%v=/path/to/your/file' and include your smashing instructions in it, e.g., \n%v", PROP_INSTRUCTION_PATH, INSTRUCTION_EXAMPLE)
	}
	return file, nil
}

func LoadInstructionFile(rail common.Rail, path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file '%v' not found", path)
		}
		return fmt.Errorf("failed to open file '%v', %v", path, err)
	}

	common.LoadConfigFromFile(path, rail)
	return nil
}

func PackSmashInstructions() SmashInstructions {
	var instructions SmashInstructions
	common.UnmarshalFromProp(&instructions)
	return instructions
}

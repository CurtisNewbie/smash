package smash

import (
	"fmt"
	"os"
	"strings"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/sirupsen/logrus"
)

type Instruction struct {
	Cron        string
	Parallelism int
	Url         string
	Method      string
	Headers     map[string]string
	Payload     string
	Curl        string
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
		return "", fmt.Errorf("please specifiy file path using '%v=/path/to/your/file' and include your smashing instructions in it", PROP_INSTRUCTION_PATH)
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

	// do some pre-processing
	copied := []Instruction{}
	for i := range instructions.Instructions {
		processed := instructions.Instructions[i]
		processed = TryParseCurl(processed)
		copied = append(copied, processed)
	}
	instructions.Instructions = copied

	return instructions
}

// TODO: improve this parser, it's now only useful for well-structured curl 'copied' from Chrome
func TryParseCurl(inst Instruction) Instruction {
	if common.IsBlankStr(inst.Curl) {
		return inst
	}
	inst.Headers = map[string]string{}
	if common.IsBlankStr(inst.Method) {
		inst.Method = "GET"
	}

	segments := curlSegments(inst.Curl)
	for i := range segments {
		seg := strings.TrimSpace(segments[i])
		// logrus.Debugf("segment %v %v", i, seg)

		if k, v, ok := parseCurlParam(seg, "-H"); ok { // header
			inst.Headers[k] = v
		} else if _, v, ok := parseCurlParam(seg, "-d"); ok { // body
			inst.Payload = v
		} else if _, v, ok := parseCurlParam(seg, "-X"); ok { // method
			inst.Method = v
		} else if v, ok := parseCurlDest(seg); ok { // destination
			inst.Url = v
		}
	}
	inst.Curl = ""
	logrus.Debugf("%+v", inst)
	return inst
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	v := []rune(s)
	if len(v) >= 2 && (v[0] == '\'' || v[0] == '"') {
		return string(v[1 : len(v)-1])
	}
	return strings.TrimSpace(string(v))
}

func parseCurlParam(seg string, prefix string) (string, string, bool) {
	if strings.HasPrefix(seg, prefix) {
		seg = unquote(string([]rune(seg)[len([]rune(prefix)):]))
		// logrus.Debugf("v: %v, prefix: %v, j: %v", seg, prefix, j)
		tokens := strings.SplitN(seg, ":", 2)
		if len(tokens) > 1 { // k : value
			k := strings.TrimSpace(tokens[0])
			v := strings.TrimSpace(tokens[1])
			// logrus.Debugf("v: %v, prefix: %v, key: %v, val: %v", seg, prefix, k, v)
			return k, v, true
		}
		if len(tokens) > 0 { // only value
			val := strings.TrimSpace(tokens[0])
			// logrus.Debugf("v: %v, prefix: %v, key: , val: %v", seg, prefix, val)
			return "", val, true
		}
	}
	return "", "", false
}

func parseCurlDest(v string) (string, bool) {
	if j := strings.Index(v, "http"); j > -1 { // it may look like 'curl "http:...." or "http:..."'
		s := []rune(v)[j:]
		logrus.Debugf("(http) s: %v, j: %v", v, j)
		k := len(s) - 1
		if s[k] == '\'' || s[k] == '"' {
			quote := s[k]
			for s[k] == quote {
				k--
			}
		}
		s = s[:k+1]
		return string(s), true
	}
	return "", false
}

func curlSegments(curl string) []string {
	// TODO: should support curl that are not so well structured
	return strings.Split(curl, "\\")
}

package main

type Steps int

const (
	StepReadConfig Steps = iota
	StepPreFetch
	StepFetch
	StepValidate
	StepProcess
	StepStore
)

func (d Steps) String() string {
	return GetSteps()[d]
}

func GetSteps() []string {
	return []string{
		"ReadConfig",
		"PreFetch",
		"Fetch",
		"Validate",
		"Process",
		"Store",
	}
}

package main

import (
	"fmt"
	"traductio/internal/inputreader"
	"traductio/internal/sink"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Input      inputreader.InputConfig `yaml:"input"`
	Validators Validators              `yaml:"validators"`
	Process    ProcessConfig           `yaml:"process"`
	Output     sink.Config             `yaml:"output"`
}

type Validators []Validator

func (v Validators) ValidateContent(data []byte) (bool, []error) {
	errs := []error{}
	ok := true
	for _, check := range v {
		v, err := queryBytes(data, check.Selector)
		if err != nil {
			ok = false
			errs = append(errs, err)
		}

		value := string(v)
			if value != check.Expect {
				ok = false
				err = fmt.Errorf("value at '%s' is expected to be '%s', is '%s'", check.Selector, check.Expect, value)
				errs = append(errs, err)
		}
	}
	return ok, errs
}

type Validator struct {
	Selector string `yaml:"selector"`
	Expect   string `yaml:"expect"`
}

type ProcessConfig struct {
	Iterator Iterator `yaml:"iterator"`
	NoTrim   bool     `yaml:"no_trim"`
}

type Iterator struct {
	Selector    string             `yaml:"selector"`
	Time        TimeSet            `yaml:"time"`
	Tags        map[string]string  `yaml:"tags"`
	FixedTags   map[string]string  `yaml:"fixed_tags"`
	Values      map[string]string  `yaml:"values"`
	FixedValues map[string]float64 `yaml:"fixed_values"`
	Iterator    *Iterator          `yaml:"iterator"`
}

type TimeSet struct {
	Selector string `yaml:"selector"`
	Format   string `yaml:"format"`
}

func (i Iterator) GetStructure() (tags []string, values []string) {
	for key := range i.Tags {
		tags = append(tags, key)
	}
	for key := range i.FixedTags {
		tags = append(tags, key)
	}
	for key := range i.Values {
		values = append(values, key)
	}
	for key := range i.FixedValues {
		values = append(values, key)
	}
	if i.Iterator != nil {
		t, v := i.Iterator.GetStructure()
		tags = append(tags, t...)
		values = append(values, v...)
	}
	return
}

func ReadConfig(cfgFile string) (Config, error) {
	c := Config{}

	i, err := inputreader.NewInput(inputreader.InputConfig{URL: cfgFile}, map[string]string{})
	if err != nil {
		return c, err
	}

	data, err := i.Fetch()
	if err != nil {
		return c, err
	}

	err = yaml.Unmarshal(data, &c)
	if err != nil {
		err = fmt.Errorf("error while parsing %s: %s", cfgFile, err.Error())
		return c, err
	}

	return c, nil
}

func (c Config) String() string {
	b, _ := yaml.Marshal(c)
	return string(b)
}

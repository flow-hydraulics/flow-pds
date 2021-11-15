package flow_helpers

import (
	"bytes"
	"io/ioutil"

	"text/template"

	"github.com/caarlos0/env/v6"
)

type CadenceTemplateVars struct {
	PDS                   string `env:"PDS_ADDRESS"`
	IPackNFT              string `env:"PDS_ADDRESS"`
	NonFungibleToken      string `env:"NON_FUNGIBLE_TOKEN_ADDRESS"`
	PackNFTName           string
	PackNFTAddress        string
	CollectibleNFTName    string
	CollectibleNFTAddress string
}

func ParseCadenceTemplate(templatePath string, vars *CadenceTemplateVars) ([]byte, error) {
	fb, err := ioutil.ReadFile(templatePath)
	if err != nil {
		panic(err)
	}

	if vars == nil {
		vars = &CadenceTemplateVars{}
	}

	if err := env.Parse(vars); err != nil {
		return nil, err
	}

	tmpl, err := template.New(templatePath).Parse(string(fb))
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}

	if err := tmpl.Execute(buf, *vars); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

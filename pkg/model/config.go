package model

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	RemoveImages    bool `json:"remove_images"`
	AdvancedEditing bool `json:"advanced_editing"`
	ShowAdd         bool `json:"show_add"`
}

func LoadConfig() (Config, error) {
	c := Config{}
	// FIXME: Use the program's dir rather than the cwd
	// dir := filepath.Dir(os.Args[0])
	dir, err := os.Getwd()
	if err != nil {
		return c, err
	}

	b, err := os.ReadFile(fmt.Sprintf("%s/pocket-editor.json", dir))
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(b, &c)
	return c, err
}

func (c Config) SaveConfig() error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	// FIXME: Use the program's dir rather than the cwd
	//dir := filepath.Dir(os.Args[0])
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	return os.WriteFile(fmt.Sprintf("%s/pocket-editor.json", dir), b, 0644)
}

package models

type Config struct {
	File_Location    string
	Work_dir         string `yaml:"work_dir"`
	File_Patterns    string `yaml:"file_patterns"`
	Exclude_Patterns string `yaml:"exclude_patterns"`
	Start_Command    string `yaml:"start_command"`
	Port             string `yaml:"port"`
}

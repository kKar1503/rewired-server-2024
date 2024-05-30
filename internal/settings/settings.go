package settings

type Settings struct {
	TCPPort uint
	WSPort  uint
	Origins string
}

func init() {
	settings = &Settings{}
}

var settings *Settings

func Get() *Settings {
	return settings
}

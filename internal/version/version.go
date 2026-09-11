package version

// Info holds version metadata.
type Info struct {
	Version string
	Commit  string
	Date    string
}

// Current returns current version metadata.
func Current(v, commit, date string) Info {
	if v == "" {
		v = "0.1.0-dev"
	}
	if commit == "" {
		commit = "none"
	}
	if date == "" {
		date = "unknown"
	}
	return Info{
		Version: v,
		Commit:  commit,
		Date:    date,
	}
}

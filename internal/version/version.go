package version

import "runtime/debug"

// Info holds version metadata.
type Info struct {
	Version string
	Commit  string
	Date    string
}

// Current returns current version metadata.
func Current(v, commit, date string) Info {
	if info, ok := debug.ReadBuildInfo(); ok {
		if (v == "" || v == "0.1.0-dev" || v == "0.2.0-dev") && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}
		if commit == "" || commit == "none" {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					commit = setting.Value
					if len(commit) > 12 {
						commit = commit[:12]
					}
					break
				}
			}
		}
		if date == "" || date == "unknown" {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.time" {
					date = setting.Value
					break
				}
			}
		}
	}
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

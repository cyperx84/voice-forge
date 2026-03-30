package tts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// PythonRuntime describes where a Python-backed TTS backend should execute.
type PythonRuntime struct {
	Name       string
	PythonPath string
	Source     string
}

func ResolveConfiguredRuntime(envVar, configuredPath, defaultDir string) PythonRuntime {
	if env := os.Getenv(envVar); env != "" {
		return pythonRuntimeFromPath(env, "env:"+envVar)
	}
	if configuredPath != "" {
		return pythonRuntimeFromPath(configuredPath, "config")
	}
	home, _ := os.UserHomeDir()
	if home != "" && defaultDir != "" {
		return pythonRuntimeFromPath(filepath.Join(home, defaultDir), "default")
	}
	return PythonRuntime{Name: "system", PythonPath: "python3", Source: "fallback"}
}

func pythonRuntimeFromPath(path, source string) PythonRuntime {
	if path == "" {
		return PythonRuntime{Name: "system", PythonPath: "python3", Source: source}
	}
	path = filepath.Clean(path)
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			py := filepath.Join(path, "bin", "python3")
			return PythonRuntime{Name: filepath.Base(path), PythonPath: py, Source: source}
		}
		return PythonRuntime{Name: filepath.Base(path), PythonPath: path, Source: source}
	}
	if filepath.Base(path) == "python3" || filepath.Base(path) == "python" {
		return PythonRuntime{Name: filepath.Base(path), PythonPath: path, Source: source}
	}
	py := filepath.Join(path, "bin", "python3")
	return PythonRuntime{Name: filepath.Base(path), PythonPath: py, Source: source}
}

func (r PythonRuntime) Available() bool {
	if r.PythonPath == "python3" || r.PythonPath == "python" {
		_, err := exec.LookPath(r.PythonPath)
		return err == nil
	}
	_, err := os.Stat(r.PythonPath)
	return err == nil
}

func (r PythonRuntime) Description() string {
	return fmt.Sprintf("%s (%s)", r.PythonPath, r.Source)
}

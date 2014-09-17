package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	errFound = errors.New("found")
)

const (
	sp      = string(os.PathListSeparator)
	version = "0.0.1"
)

func main() {
	var (
		paths      []string
		searchDirs []string
	)

	searchProgramFiles := flag.Bool("p", false, "Search Program Files directories")
	showAll := flag.Bool("a", false, "Show all matches")
	help := flag.Bool("h", false, "Show this help message")
	printVersion := flag.Bool("v", false, "Print version and exit")
	flag.Usage = usage

	flag.Parse()

	commands := flag.Args()

	if *printVersion {
		fmt.Fprintf(os.Stdout, "%s %v, Copyright (C) 2014 Sol Touré.\n", filepath.Base(os.Args[0]), version)
		return
	}

	if *help || len(commands) == 0 {
		flag.Usage()
		return
	}

	//Add the current directory to the search path
	if currentDir, err := filepath.Abs(filepath.Dir(os.Args[0])); err == nil {
		paths = append(paths, currentDir)
	}

	if *searchProgramFiles {
		searchDirs = append(searchDirs, strings.Split(os.Getenv("ProgramFiles"), sp)...)
		searchDirs = append(searchDirs, strings.Split(os.Getenv("ProgramFiles(x86)"), sp)...)
	}

	paths = append(paths, strings.Split(os.Getenv("PATH"), sp)...)
	pathExt := map[string]struct{}{
		".com": struct{}{},
		".exe": struct{}{},
		".bat": struct{}{},
		".cmd": struct{}{},
	}

	for _, ext := range strings.Split(os.Getenv("PATHEXT"), sp) {
		pathExt[strings.ToLower(ext)] = struct{}{}
	}

	result := newResultMap()

	for _, p := range paths {
		for _, cmd := range commands {
			f := filepath.Join(p, cmd)
			if !result.hasKey(cmd) || *showAll {

				if len(filepath.Ext(cmd)) == 0 {
					for ext, _ := range pathExt {
						name := f + ext
						if (!result.hasKey(cmd) || *showAll) && isFile(name) && result.add(cmd, name) {
							fmt.Println(name)
						}
					}
				} else if (!result.hasKey(cmd) || *showAll) && isFile(f) && result.add(cmd, f) {
					fmt.Println(f)
				}
			}
		}
	}

	if *showAll || len(result) != len(commands) {
		for _, d := range searchDirs {
			walk(d, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return filepath.SkipDir
				}

				if info.IsDir() {
					return nil
				}

				for _, cmd := range commands {
					found := result.hasKey(cmd)
					addExtension := len(filepath.Ext(cmd)) == 0

					if !found || *showAll {
						base := filepath.Base(p)
						if _, ok := pathExt[strings.ToLower(filepath.Ext(p))]; ok {
							name := cmd
							if addExtension {
								name += filepath.Ext(p)
							}

							if strings.ToLower(base) == strings.ToLower(name) && result.add(cmd, p) {
								fmt.Println(p)
							}
						}
					}
					if !*showAll && len(commands) == len(result) {
						return errFound
					}
				}

				if !*showAll && len(commands) == len(result) {
					return errFound
				}

				return nil
			})
		}
	}

	for _, cmd := range commands {
		if _, found := result[cmd]; !found {
			fmt.Printf("\n%s: no %q in %v\n\n", filepath.Base(os.Args[0]), cmd, append(paths, searchDirs...))
		}
	}
	fmt.Printf("%v\n", result)
}

type resultMap map[string]map[string]struct{}

func newResultMap() resultMap {
	return make(map[string]map[string]struct{})
}

func (r resultMap) add(key, value string) bool {
	if _, ok := r[key]; ok {
		if _, ok := r[key][value]; ok {
			return !ok
		}
		r[key][value] = struct{}{}
		return ok
	}
	r[key] = make(map[string]struct{})
	r[key][value] = struct{}{}
	return true
}

func (r resultMap) hasKey(key string) bool {
	_, ok := r[key]
	return ok
}

func isFile(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && !stat.IsDir()
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] COMMAND [...]\n", filepath.Base(os.Args[0]))
	flag.PrintDefaults()
}

type FileInfo struct {
	Path string
	os.FileInfo
}

func readdir(path string) ([]os.FileInfo, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	return f.Readdir(-1)
}

func walk(path string, walkFn filepath.WalkFunc) error {
	var (
		stack   []*FileInfo
		current *FileInfo
	)

	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() {
		return walkFn(path, info, err)
	}

	stack = append(stack, &FileInfo{path, info})

	for pos := len(stack) - 1; pos > -1; pos = len(stack) - 1 {
		current, stack = stack[pos], stack[:pos]

		if err := walkFn(current.Path, current, nil); err != nil {
			if err != filepath.SkipDir {
				return err
			}
			continue
		}

		infos, _ := readdir(current.Path)

		for _, info := range infos {
			sub := filepath.Join(current.Path, info.Name())

			if info.IsDir() {
				stack = append(stack, &FileInfo{sub, info})
			} else if err := walkFn(sub, info, nil); err != nil && err != filepath.SkipDir {
				return err
			}
		}
	}

	return nil
}

package main

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Fileinfo is used for file info
type Fileinfo struct {
	Name         string
	Level        int
	Path         string
	Parent       string
	ParentNode   *Fileinfo
	Size         string
	IsDir        bool
	Last         bool
	Children     []*Fileinfo
	Check        bool
	neighborhood map[string]*Fileinfo
	output       string
}

// SortByPath is used for sorting
type SortByPath []*Fileinfo

func (paths *SortByPath) sortPaths() *SortByPath {
	filename := func(c1, c2 *Fileinfo) bool {
		return c1.Name < c2.Name
	}
	level := func(c1, c2 *Fileinfo) bool {
		return c1.Level < c2.Level
	}
	OrderedBy(level, filename).Sort(*paths)

	return paths
}

func (paths *SortByPath) detectRelativeNodes() *SortByPath {
	PathList := *paths
	for i, item := range PathList {
		if item.IsDir == true {
			parentDirectory := PathList[i]
			parentDirectoryPath := strings.Trim(parentDirectory.Parent+string(os.PathSeparator)+parentDirectory.Name, string(os.PathSeparator))

			for j, subItem := range PathList {
				if subItem.Parent == parentDirectoryPath {
					child := PathList[j]
					parentDirectory.Children = append(parentDirectory.Children, child)
					child.ParentNode = parentDirectory
				}
			}
		}
	}

	return paths
}

func (paths *SortByPath) defineNeighborhoods() *SortByPath {
	PathList := *paths
	for i := range PathList {
		PathList.defineNodeNeighborhood(PathList[i])
	}

	return &PathList
}

func (paths *SortByPath) defineNodeNeighborhood(CurrentNode *Fileinfo) *Fileinfo {
	PathList := *paths

	var parentDirectory = CurrentNode.ParentNode
	var parentDirectoryPath string

	if parentDirectory != nil {
		parentDirectoryPath = strings.Trim(parentDirectory.Parent+string(os.PathSeparator)+parentDirectory.Name, string(os.PathSeparator))
	}

	var neighborhood map[string]*Fileinfo = map[string]*Fileinfo{
		"prev": nil,
		"next": nil,
	}

	var foundTrigger = false

	for j, item := range PathList {
		if item.Parent == parentDirectoryPath {
			if item.Path == CurrentNode.Path {
				foundTrigger = true
			} else if foundTrigger == false {
				neighborhood["prev"] = PathList[j]
			} else if foundTrigger == true {
				neighborhood["next"] = PathList[j]
				break
			}
		}
	}

	CurrentNode.neighborhood = neighborhood
	return CurrentNode
}

func (paths *SortByPath) generateNodeOutputs() *SortByPath {
	PathList := *paths
	for i := range PathList {
		PathList.generateNodeOutput(PathList[i])
	}

	return &PathList
}

func (paths *SortByPath) generateNodeOutput(CurrentNode *Fileinfo) *Fileinfo {
	node := CurrentNode
	var output string
	if CurrentNode.neighborhood["next"] == nil {
		output = "└───"
	} else {
		output = "├───"
	}
	output += node.Name + node.Size + "\n"

	for true {
		node = node.ParentNode
		if node == nil {
			break
		}

		tab := ""
		if node.neighborhood["next"] == nil {
			tab = "\t"
		} else {
			tab = "│\t"
		}
		output = tab + output
	}

	CurrentNode.output = output

	return CurrentNode
}

type lessFunc func(p1, p2 *Fileinfo) bool

// multiSorter implements the Sort interface, sorting the changes within.
type multiSorter struct {
	changes []*Fileinfo
	less    []lessFunc
}

// Sort sorts the argument slice according to the less functions passed to OrderedBy.
func (ms *multiSorter) Sort(changes []*Fileinfo) {
	ms.changes = changes
	sort.Sort(ms)
}

// OrderedBy returns a Sorter that sorts using the less functions, in order.
// Call its Sort method to sort the data.
func OrderedBy(less ...lessFunc) *multiSorter {
	return &multiSorter{
		less: less,
	}
}

// Len is part of sort.Interface.
func (ms *multiSorter) Len() int {
	return len(ms.changes)
}

// Swap is part of sort.Interface.
func (ms *multiSorter) Swap(i, j int) {
	ms.changes[i], ms.changes[j] = ms.changes[j], ms.changes[i]
}

// Less is part of sort.Interface. It is implemented by looping along the
// less functions until it finds a comparison that discriminates between
// the two items (one is less than the other). Note that it can call the
// less functions twice per call. We could change the functions to return
// -1, 0, 1 and reduce the number of calls for greater efficiency: an
// exercise for the reader.
func (ms *multiSorter) Less(i, j int) bool {
	p, q := &ms.changes[i], &ms.changes[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		switch {
		case less(*p, *q):
			// p < q, so we have a decision.
			return true
		case less(*q, *p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports
	return ms.less[k](*p, *q)
}

func getPathList(path string, printFiles bool) SortByPath {
	var PathList []*Fileinfo

	var DefaultFileInfo []*Fileinfo
	var FInfo *Fileinfo
	filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err == nil {

				if printFiles == false && info.IsDir() == false {
					return nil
				}

				var pathSlice = strings.Split(func(path *string) string {
					*path = strings.Trim(*path, "."+string(os.PathSeparator))
					return strings.Trim(*path, string(os.PathSeparator))
				}(&path), string(os.PathSeparator))

				if len(pathSlice) == 1 {
					return nil
				}

				PathList = append(PathList, &Fileinfo{
					info.Name(),
					len(pathSlice) - 1,
					path,
					func(pathSlice []string) string {
						parent := ""
						for i := 1; i < len(pathSlice)-1; i++ {
							parent = parent + pathSlice[i] + string(os.PathSeparator)
						}
						return strings.Trim(parent, string(os.PathSeparator))
					}(pathSlice),
					FInfo,
					func(size int64, isDir bool) string {
						if isDir {
							return ""
						}
						if size > 0 {
							return " (" + strconv.FormatInt(size, 10) + "b)"
						}
						return " (empty)"
					}(info.Size(), info.IsDir()),
					info.IsDir(),
					false,
					DefaultFileInfo,
					false,
					make(map[string]*Fileinfo, 2),
					"",
				})
			}

			return nil
		})

	return PathList
}

func outputChildren(nodes []*Fileinfo, output *string) {
	for _, item := range nodes {
		*output += item.output
		outputChildren(item.Children, output)
	}
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	PathList := getPathList(path, printFiles)
	PathList.sortPaths().detectRelativeNodes().defineNeighborhoods().generateNodeOutputs()

	outputList := PathList

	var output string

	var filteredTree SortByPath

	for i, item := range outputList {
		if item.Level == 1 {
			filteredTree = append(filteredTree, outputList[i])
		}
	}

	outputChildren(filteredTree, &output)
	out.Write([]byte(output))

	//fmt.Println(output)
	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"

	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

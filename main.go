package main

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func lsmodLineToKernelModule(index int, line string) KernelModule {
	spaceRegexp := regexp.MustCompile(`\s+`)
	line = spaceRegexp.ReplaceAllString(line, " ")

	lineElements := strings.Split(line, " ")

	var kernel_module KernelModule
	for elementIndex, element := range lineElements {
		if elementIndex == 0 {
		}
		switch elementIndex {
		case 0:
			kernel_module.Name = element
			break
		case 2:
			kernel_module.UsedByCount, _ = strconv.ParseInt(element, 10, 64)
			break
		case 3:
			kernel_module.UsedBySlice = strings.Split(element, ",")
		}
	}
	kernel_module.Id = index
	return kernel_module
}

func runLsmod() []string {
	cmd := "lsmod"
	lsmodOutputByte, _ := exec.Command("bash", "-c", cmd).Output()
	lsmodOutput := string(lsmodOutputByte)
	lsmodLineByLine := strings.Split(lsmodOutput, "\n")
	return lsmodLineByLine[1:]
}

func readLsmod(maxModulesOptional ...int) []KernelModule {
	var maxModules int = math.MaxInt32
	if len(maxModulesOptional) > 0 {
		maxModules = maxModulesOptional[0]
	}
	var kernelModules []KernelModule
	lsmodLineByLine := runLsmod()
	for i, line := range lsmodLineByLine[:len(lsmodLineByLine)-1] {
		if i >= maxModules {
			break
		}
		kernelModules = append(kernelModules, lsmodLineToKernelModule(i+1, line))
	}
	return kernelModules
}

type KernelModule struct {
	Name        string
	UsedByCount int64
	UsedBySlice []string
	Id          int
}

func get_nodes_string(modules []KernelModule) string {
	var res string = "\nvar nodes = new vis.DataSet(["
	for _, module := range modules {
		res += "\n\t{id: " + strconv.Itoa(module.Id) + ", label: '" + module.Name + "'},"
		var n Node = Node{module.Id, module.Name}
		nodes = append(nodes, n)
	}
	res += "\n]);"
	return res
}

func get_edges_string(modules []KernelModule) string {
	var res string = "\nvar edges = new vis.DataSet(["
	edges_set := map[string]string{}
	for _, module := range modules {
		for _, usedByItem := range module.UsedBySlice {
			edges_set[usedByItem] = module.Name

			res += "\n\t{from: " + strconv.Itoa(name_to_id[usedByItem]) + ", to: " + strconv.Itoa(name_to_id[module.Name]) + ", arrows:'to'},"
			var e Edge = Edge{name_to_id[usedByItem], name_to_id[module.Name]}
			edges = append(edges, e)
		}
	}
	res += `
	]);`
	return res
}

func create_html(nodes_and_edges string) string {
	var html string = `
<!doctype html>
<html>
<head>
  <title>lsmod graph</title>
  <script type="text/javascript" src="./vis/vis.js"></script>
  <link href="./vis/vis-network.min.css" rel="stylesheet" type="text/css" />
  <style type="text/css">
    #mynetwork {
      width: 70%;
      height: 800px;
  margin: auto;
  border: 3px solid green;
  padding: 10px;
}
    }
  </style>
</head>
<body>
<p>
  lsmod:
</p>
<div id="mynetwork"></div>
<script type="text/javascript">
  // create an array with nodes
`
	html += nodes_and_edges
	html += `
// create a network
  var container = document.getElementById('mynetwork');
  var data = {
    nodes: nodes,
    edges: edges
  };
    var options = {
  };
  var network = new vis.Network(container, data, options);
</script>
</body>
</html>
`
	return html
}

func create_html_file(fileName string, htmlString string) {
	f, err := os.Create(fileName)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
	f.WriteString(htmlString)
	f.Sync()
}

var id_to_name = make(map[int]string)
var name_to_id = make(map[string]int)

type Node struct {
	Id    int
	Label string
}

type MyData struct {
	Nodes []Node
	Edges []Edge
}

type Edge struct {
	From int
	To   int
	//Arrows string
}

var nodes []Node
var edges []Edge

func set_maps(modules []KernelModule) {
	for _, module := range modules {
		id_to_name[module.Id] = module.Name
		name_to_id[module.Name] = module.Id
	}
}

func http_server1(nodes_and_edges string, port int) {
	tmpl := template.Must(template.ParseFiles("./layout.html"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data :=
			struct {
				NodesAndEdges string
				Edges         string
			}{
				NodesAndEdges: nodes_and_edges,
			}
		tmpl.Execute(w, data)
	})
	port_str := ":" + strconv.Itoa(port)
	http.ListenAndServe(port_str, nil)
}

func http_server(port int) {
	fs := http.FileServer(http.Dir("vis"))
	http.Handle("/vis/", http.StripPrefix("/vis/", fs))
	tmpl := template.Must(template.ParseFiles("./layout.html"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := MyData{
			Nodes: nodes,
			Edges: edges,
		}
		tmpl.Execute(w, data)
	})
	port_str := ":" + strconv.Itoa(port)
	err := http.ListenAndServe(port_str, nil)
	fmt.Println(err)
}

func main() {
	modules := readLsmod(20)
	set_maps(modules)

	nodes_and_edges := get_nodes_string(modules)
	nodes_and_edges += get_edges_string(modules)
	htmlString := create_html(nodes_and_edges)
	create_html_file("lsmod.html", htmlString)
	http_server(8080)
}

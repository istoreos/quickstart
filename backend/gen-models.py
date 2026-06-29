from os import walk, system, remove, path, environ
import codecs

def process_lines(lines):
    i = 0
    outs = ["package models\n", "\n"]
    while i < len(lines):
        line = lines[i]
        if line.startswith("type "):
            start = find_start(lines, i)
            end = i
            if line.strip("\n").endswith(" struct {"):
                end = find_end(lines, i)
            outs.extend(lines[start:end+1])
            #sub_lines = "".join(lines[start:end+1])
            #print sub_lines
            i = end
        i = i+1
    return outs

def find_end(lines, i):
    for j in range (i+1, len(lines)):
        if lines[j].strip("\n") == "}":
            return j

def find_start(lines, i):
    line = lines[i]
    for j in range(i-1, -1, -1):
        l2 = lines[j]
        if not l2.startswith("// "):
            return j+1
    return i

mypath = "models"
target_path = "models"

if "YAML" in environ:
    yaml_file = environ["YAML"]
else:
    yaml_file = 'yamls/quickstart.yaml'

system('swagger generate model --spec='+yaml_file)
path_to_remove = ["/unix_drive_do_mount.go"]
for p in path_to_remove:
    if path.exists(mypath + p):
        remove(mypath + "/unix_drive_do_mount.go")

path_to_ignore = {"file_basic_sort.go"}

f = []
for (dirpath, dirnames, filenames) in walk(mypath):
    f.extend(filenames)
    break

for one_path in f:
    one_file = codecs.open(mypath + "/" + one_path, "r", "utf-8")
    if one_path in path_to_ignore:
        continue
    #print one_path
    lines = one_file.readlines()
    outs = process_lines(lines)
    one_file.close()
    with codecs.open(target_path + "/" + one_path, "w", "utf-8") as f2:
        f2.write("".join(outs))

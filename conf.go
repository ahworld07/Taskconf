package Taskconf

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"github.com/ahworld07/Taskutil"
	"github.com/go-ini/ini"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

var PodConfig string = `[templetes]
pod_name = bcl2fq-all
module = qc
template = bcl2fq
imagePullSecrets = registry-read-only-key-yw

[Tolerations.network]
Operator = Equal
Value = internet
Effect = NoSchedule

[NodeSelector]
env = idc_physical

[requests]
memory = 1
cpu = 1

[limits]
memory = 1
cpu = 1

[container]
image = registry-vpc.cn-hangzhou.aliyuncs.com/annoroad/annogene-base:v0.1
args = /home/zanyuan/bin/example/submit_test/bcl2fq.sh
lines = 
[volumeMounts]
home = /cluster_home|store|/home
cloud = /cloud|store|/annogene/cloud
datayw = /datayw|store|/annogene/datayw
`

//This struct is used to read/write the config file, including default parameters and project database information.
type ConfigFile struct {
	Conffile    string
	Cfg         *ini.File
}

func InitGomonitor(){
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	Taskutil.CheckErr(err)
	GMconfFile := filepath.Join(dir, "gomonitor.conf")
	exit_file, err := Taskutil.PathExists(GMconfFile)
	Taskutil.CheckErr(err)

	if exit_file == false {
		f, _ := os.OpenFile(GMconfFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
		defer f.Close()
	}

	cfg, _ := ini.LoadSources(ini.LoadOptions{AllowBooleanKeys: true}, GMconfFile)

	_ ,_ = cfg.NewSection("base")
	_ ,_ = cfg.NewSection("kubectl")
	_ ,_ = cfg.NewSection("volumeMounts")

	Taskutil.CheckErr(err)
	_, err = cfg.Section("base").NewKey("defaultFinishMark","Still_waters_run_deep")
	Taskutil.CheckErr(err)
	_, err = cfg.Section("kubectl").NewKey("RunAsGroup","511")
	Taskutil.CheckErr(err)

	_, err = cfg.Section("kubectl").NewKey("imagePullPolicy","IfNotPresent")
	Taskutil.CheckErr(err)

	_, err = cfg.Section("kubectl").NewKey("imageRegistry","registry-vpc.cn-hangzhou.aliyuncs.com/annoroad/")
	Taskutil.CheckErr(err)
	_, err = cfg.Section("kubectl").NewKey("image","annogene-base:v0.1")
	Taskutil.CheckErr(err)

	_, err = cfg.Section("kubectl").NewKey("NodeSelector","env:idc_physical")
	Taskutil.CheckErr(err)

	_, err = cfg.Section("kubectl").NewKey("imagePullSecrets","registry-read-only-key-yw")
	Taskutil.CheckErr(err)

	_, err = cfg.Section("volumeMounts").NewKey("home","/cluster_home|store|/home")
	Taskutil.CheckErr(err)
	_, err = cfg.Section("volumeMounts").NewKey("cloud","/cloud|store|/annogene/cloud")
	Taskutil.CheckErr(err)

	err = cfg.SaveTo(GMconfFile)
	Taskutil.CheckErr(err)

	Taskutil.WriteWithIoutil(filepath.Join(dir, "example.submit.ini"), PodConfig)
}

func GetDefault()(defultCfg *ini.File){
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	Taskutil.CheckErr(err)
	GMconfFile := filepath.Join(dir, "gomonitor.conf")

	defultCfg, _ = ini.LoadSources(ini.LoadOptions{AllowBooleanKeys: true}, GMconfFile)

	return
}

func (cff *ConfigFile)SetDefault(){
	Hname, err := os.Hostname()
	CheckErr(err)

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	Taskutil.CheckErr(err)
	GMconfFile := filepath.Join(dir, "gomonitor.conf")
	exit_file, err := Taskutil.PathExists(GMconfFile)
	Taskutil.CheckErr(err)

	if exit_file == false {
		f, _ := os.OpenFile(GMconfFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
		defer f.Close()
	}

	default_cfg, _ := ini.LoadSources(ini.LoadOptions{AllowBooleanKeys: true}, GMconfFile)

	_ ,_ = cff.Cfg.NewSection("project")
	_ ,_ = cff.Cfg.NewSection("base")
	_ ,_ = cff.Cfg.NewSection("kubectl")
	_ ,_ = cff.Cfg.NewSection("volumeMounts")

	_, err = cff.Cfg.Section("base").NewKey("CronNode",Hname)
	CheckErr(err)
	_, err = cff.Cfg.Section("base").NewKey("pobMaxRetries","3")
	CheckErr(err)

	RunAsGroup := cff.Cfg.Section("kubectl").Key("RunAsGroup").String()
	if RunAsGroup == ""{
		gid := default_cfg.Section("kubectl").Key("RunAsGroup").String()
		user, _ := user.Current()
		if user.Name == "sci-qc"{
			gid = "674"
		}
		_, err = cff.Cfg.Section("kubectl").NewKey("RunAsGroup",gid)
		CheckErr(err)
	}


	SetDefaultConf(cff, default_cfg, "base","defaultFinishMark")
	SetDefaultConf(cff, default_cfg, "kubectl", "imagePullPolicy")
	SetDefaultConf(cff, default_cfg, "kubectl", "imageRegistry")
	SetDefaultConf(cff, default_cfg, "kubectl", "image")
	SetDefaultConf(cff, default_cfg, "kubectl", "NodeSelector")
	SetDefaultConf(cff, default_cfg, "kubectl", "imagePullSecrets")
	SetDefaultConf(cff, default_cfg, "kubectl", "NodeSelector")

	volumeMounts, _ := default_cfg.GetSection("volumeMounts")
	for _,key := range volumeMounts.Keys(){
		value := cff.Cfg.Section("volumeMounts").Key(key.String()).String()
		if value == ""{
			_, err = cff.Cfg.Section("volumeMounts").NewKey(key.String(),key.Value())
			CheckErr(err)
		}
	}

	cff.Update()
}

func SetDefaultConf(cff *ConfigFile, default_cfg *ini.File,section, key string){
	value := cff.Cfg.Section(section).Key(key).String()
	if value == ""{
		value = default_cfg.Section(section).Key(key).String()
		_, err := cff.Cfg.Section(section).NewKey(key, value)
		CheckErr(err)
	}
}


func Config_Init()(cff *ConfigFile){
	home, _ := Home()
	Conf_file := path.Join(home, "gomonitor.project.conf")
	exit_file, err := PathExists(Conf_file)
	CheckErr(err)
	needupdate := false
	if exit_file == false {
		f, _ := os.OpenFile(Conf_file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
		defer f.Close()
		needupdate = true
	}
	cfg, err := ini.LoadSources(ini.LoadOptions{AllowBooleanKeys: true}, Conf_file)
	CheckErr(err)

	pobMaxRetries := cfg.Section("base").Key("pobMaxRetries").String()
	RunAsGroup := cfg.Section("kubectl").Key("RunAsGroup").String()
	imagePullPolicy := cfg.Section("kubectl").Key("imagePullPolicy").String()
	imageRegistry := cfg.Section("kubectl").Key("imageRegistry").String()
	image := cfg.Section("kubectl").Key("image").String()
	NodeSelector := cfg.Section("kubectl").Key("NodeSelector").String()
	imagePullSecrets := cfg.Section("kubectl").Key("imagePullSecrets").String()

	if pobMaxRetries == "" || RunAsGroup == "" || imagePullPolicy == "" || imageRegistry == "" || image == "" || NodeSelector == "" || imagePullSecrets == ""{
		needupdate = true
	}

	cff = &ConfigFile{Conffile:Conf_file, Cfg:cfg}
	if needupdate{
		cff.SetDefault()
	}
	return
}

/*
func Programe_conf(bin string)(cfg *ini.File){
	Conf_file := path.Join(bin, "gomonitor.ini")
	exit_file, err := DAG2yaml.PathExists(Conf_file)
	DAG2yaml.CheckErr(err)
	if exit_file == false {
		panic(fmt.Sprintf("%s not exists!", Conf_file))
	}
	cfg, err = ini.LoadSources(ini.LoadOptions{AllowBooleanKeys: true}, Conf_file)
	return
}
*/

func (cff *ConfigFile)AddPrj(prjName, ProjectType, ProjectBatch, WorkFlowMode, prjdb string, GM_projects_DBconn *sql.DB){
	/*gomonitor_v0.11
	_, err := cff.Cfg.Section("project").NewKey(prjname, prjdb)
	CheckErr(err)
	*/
	stmt, err := GM_projects_DBconn.Prepare("INSERT INTO projects(ProjectName, ProjectType, ProjectBatch, WorkFlowMode, DbPath, Status, IsUpdateNow) values(?,?,?,?,?,?,?)")
	CheckErr(err)
	rows, err := GM_projects_DBconn.Query("select ProjectName from projects where ProjectName = ? and ProjectType = ? and ProjectBatch = ? and WorkFlowMode = ?", prjName, ProjectType, ProjectBatch, WorkFlowMode)
	if CheckCount(rows)==0 {
		_, err = stmt.Exec(prjName, ProjectType, ProjectBatch, WorkFlowMode, prjdb, "Unsubmit", "no")
		CheckErr(err)
	}
}


func (cff *ConfigFile)Update(){
	err := cff.Cfg.SaveTo(cff.Conffile)
	CheckErr(err)
}

func (cff *ConfigFile)RemovePrj(prjname string){
	prjdb := cff.Cfg.Section("project").Key(prjname).String()
	exit_file, err := PathExists(prjdb)
	CheckErr(err)
	if exit_file == true {
		err = os.Remove(prjdb)
		CheckErr(err)
	}
	cff.Cfg.Section("project").DeleteKey(prjname)
}

func Home() (string, error) {
	user, err := user.Current()
	if nil == err {
		return user.HomeDir, nil
	}

	// cross compile support

	if "windows" == runtime.GOOS {
		return homeWindows()
	}

	// Unix-like system, so just assume Unix
	return homeUnix()
}

func homeUnix() (string, error) {
	// First prefer the HOME environmental variable
	if home := os.Getenv("HOME"); home != "" {
		return home, nil
	}

	// If that fails, try the shell
	var stdout bytes.Buffer
	cmd := exec.Command("sh", "-c", "eval echo ~$USER")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		return "", errors.New("blank output when reading home directory")
	}

	return result, nil
}

func homeWindows() (string, error) {
	drive := os.Getenv("HOMEDRIVE")
	path := os.Getenv("HOMEPATH")
	home := drive + path
	if drive == "" || path == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		return "", errors.New("HOMEDRIVE, HOMEPATH, and USERPROFILE are blank")
	}

	return home, nil
}

func CheckErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (CronL *CronList)AddCronfile(addstr string){
	home, _ := Home()
	cronfile := path.Join(home, "gomonitor.addCrontab")
	for {
		exit_file, err := PathExists(cronfile)
		CheckErr(err)
		if exit_file == false{
			break
		}else{
			cronfile = cronfile + "1"
		}
	}

	f, err := os.Create(cronfile)
	fmt.Println(err)
	defer f.Close()
	f.WriteString(addstr)

	cmd1 := exec.Command("sh","-c", fmt.Sprintf("crontab %s", cronfile))
	_ = cmd1.Run()
	cmd2 := exec.Command("sh","-c", fmt.Sprintf("rm %s", cronfile))
	_ = cmd2.Run()
}

type CronList struct {
	Program    string
}

func (CronL *CronList)AddCron(cff *ConfigFile){
	oldnode := cff.Cfg.Section("base").Key("CronNode").String()
	curnode, _:= os.Hostname()

	if oldnode != curnode{
		fmt.Println(fmt.Sprintf("Warning: You have monitor jobs on node %s\nIf you want to work on current node,\nplease use %s cron -m 5 to change.", oldnode, CronL.Program))
		return
	}

	needAddCron := 0
	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command("sh","-c","crontab -l")
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	_ = cmd.Run()
	stdout := outbuf.String()
	tmp := strings.Split(stdout,"\n")

	if strings.Index("gomonitor", stdout) == 0{
		needAddCron = 1
	}

	if needAddCron == 1{
		addstr := fmt.Sprintf("5-59/10 * * * * %s cron -m 1\n0 0 1 * * %s cron -m 2", CronL.Program, CronL.Program)
		if len(tmp) != 0{
			addstr = addstr + "\n" + stdout
		}
		CronL.AddCronfile(addstr)
	}
}

func (CronL *CronList)RemoveCron(cff *ConfigFile){
	oldnode := cff.Cfg.Section("base").Key("CronNode").String()
	curnode, _:= os.Hostname()

	if oldnode != curnode{
		fmt.Println(fmt.Sprintf("Warning: You have monitor jobs on node %s\nIf you want to work on current node,\nplease use %s cron -m 5 to change.", oldnode, CronL.Program))
		return
	}

	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command("sh","-c","crontab -l")
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	_ = cmd.Run()
	stdout := outbuf.String()
	tmp := strings.Split(stdout,"\n")

	addstr := ""
	for _, line := range tmp{
		if strings.Index("gomonitor", line) == 0{
			addstr = addstr + "\n" + line
		}
	}
	if addstr != ""{
		CronL.AddCronfile(addstr)
	}
}


func (CronL *CronList)CheckCron(cff *ConfigFile)(node bool){
	node = true
	oldnode := cff.Cfg.Section("base").Key("CronNode").String()
	curnode, _:= os.Hostname()

	if oldnode != curnode{
		fmt.Println(fmt.Sprintf("Warning: You have monitor jobs on node %s\nIf you want to work on current node,\nplease use %s cron -m 5 to change.", oldnode, CronL.Program))
		node = false
	}
	return
}

func (CronL *CronList)ChangeCron(cff *ConfigFile){
	curnode, _:= os.Hostname()
	_, err := cff.Cfg.Section("base").NewKey("CronNode", curnode)
	CheckErr(err)
	cff.Update()
	return
}

func Crt_gm_project_tb(Db *sql.DB){
	sql_projects_table := `
	CREATE TABLE IF NOT EXISTS projects(
		Id INTEGER NOT NULL PRIMARY KEY,
		ProjectName TEXT,
		ProjectType	TEXT,
		ProjectBatch	TEXT,
		WorkFlowMode	TEXT,
		DbPath	TEXT,
		Status	TEXT,
		IsUpdateNow	TEXT,
		Start_time	datetime,
		End_time	datetime
	);
	`
	_, err := Db.Exec(sql_projects_table)
	if err != nil { panic(err) }
}

func CheckCount(rows *sql.Rows) (count int) {
	count = 0
	for rows.Next() {
		count ++
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	return count
}

/*
func Cff_Projects2DB(cff *ConfigFile, Db *sql.DB){
	stmt, err := Db.Prepare("INSERT INTO projects(ProjectName, ProjectType, ProjectBatch, WorkFlowMode, DbPath, IsUpdateNow) values(?,?,?,?,?,?)")
	CheckErr(err)

	ProjectType := "Unknown"
	user, _ := user.Current()
	if user.Name == "filter"{
		ProjectType = "filter"
	}
	if user.Name == "sci-qc"{
		ProjectType = "splite"
	}

	ProjectBatch := "None"

	for prjName, dbpath := range cff.Cfg.Section("project").KeysHash() {
		rows, err := Db.Query("select ProjectName from projects where ProjectName = ?", prjName)
		if CheckCount(rows)==0 {
			if ProjectType == "filter"{
				ProjectBatch = strings.Split(prjName, "_")[1]
			}
			_, err = stmt.Exec(prjName, ProjectType, ProjectBatch, "taskmonitor", dbpath, "no")
			CheckErr(err)
		}

		cff.Cfg.Section("project").DeleteKey(prjName)
	}
	cff.Update()
}
*/


func Creat_project_DB(cff *ConfigFile)(conn *sql.DB){
	home, _ := Home()
	GM_dbfile := path.Join(home, ".gomonitor.project.db")
	exit_file, err := PathExists(GM_dbfile)

	if exit_file == false {
		os.Create(GM_dbfile)
	}
	//db init
	//create table
	conn, err = sql.Open("sqlite3", GM_dbfile)
	CheckErr(err)

	Crt_gm_project_tb(conn)

	//后期删除此项,所有project均在
	//Cff_Projects2DB(cff, conn)

	return
}



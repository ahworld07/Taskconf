package Taskconf

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-ini/ini"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"runtime"
	"strings"
)

//This struct is used to read/write the config file, including default parameters and project database information.
type ConfigFile struct {
	Conffile    string
	Cfg         *ini.File
}

func (cff *ConfigFile)SetDefault(){
	Hname, err := os.Hostname()
	CheckErr(err)

	_ ,_ = cff.Cfg.NewSection("project")
	_ ,_ = cff.Cfg.NewSection("base")
	_ ,_ = cff.Cfg.NewSection("kubectl")
	_ ,_ = cff.Cfg.NewSection("volumeMounts")

	_, err = cff.Cfg.Section("base").NewKey("CronNode",Hname)
	CheckErr(err)
	_, err = cff.Cfg.Section("base").NewKey("defaultFinishMark","Still_waters_run_deep")
	CheckErr(err)
	_, err = cff.Cfg.Section("base").NewKey("pobMaxRetries","3")
	CheckErr(err)

	RunAsGroup := cff.Cfg.Section("kubectl").Key("RunAsGroup").String()
	if RunAsGroup == ""{
		gid := "511"
		user, _ := user.Current()
		if user.Name == "sci-qc"{
			gid = "674"
		}
		_, err = cff.Cfg.Section("kubectl").NewKey("RunAsGroup",gid)
		CheckErr(err)
	}

	imagePullPolicy := cff.Cfg.Section("kubectl").Key("imagePullPolicy").String()
	if imagePullPolicy == ""{
		_, err = cff.Cfg.Section("kubectl").NewKey("imagePullPolicy","Always")
		CheckErr(err)
	}

	_, err = cff.Cfg.Section("kubectl").NewKey("imageRegistry","registry-vpc.cn-hangzhou.aliyuncs.com/annoroad/")
	CheckErr(err)
	_, err = cff.Cfg.Section("kubectl").NewKey("image","annogene-base:v0.1")
	CheckErr(err)

	NodeSelector := cff.Cfg.Section("kubectl").Key("imagePullPolicy").String()
	if NodeSelector == ""{
		_, err = cff.Cfg.Section("kubectl").NewKey("NodeSelector","env:idc_physical")
		CheckErr(err)
	}

	imagePullSecrets := cff.Cfg.Section("kubectl").Key("imagePullSecrets").String()
	if imagePullSecrets == ""{
		_, err = cff.Cfg.Section("kubectl").NewKey("imagePullSecrets","registry-read-only-key-yw")
		CheckErr(err)
	}

	_, err = cff.Cfg.Section("volumeMounts").NewKey("home","/cluster_home|store|/home")
	CheckErr(err)
	_, err = cff.Cfg.Section("volumeMounts").NewKey("cloud","/cloud|store|/annogene/cloud")
	CheckErr(err)

	cff.Update()
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

func (cff *ConfigFile)AddPrj(prjname, prjdb string){
	_, err := cff.Cfg.Section("project").NewKey(prjname, prjdb)
	CheckErr(err)
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

/*
func Creat_tb(cff *Taskconf.ConfigFile, ProjectObj *Project)(dbObj *MySql){
	home, _ := Home()
	Conf_file := path.Join(home, ".gomonitor.project.db")
	exit_file, err := PathExists(Conf_file)

	exit_file, _ := DAG2yaml.PathExists(dbpath)
	if exit_file == false {
		//_ = os.Remove(dbpath)
		os.Create(dbpath)
	}

	//db init
	//create table
	conn, err := sql.Open("sqlite3", dbpath)
	DAG2yaml.CheckErr(err)
	dbObj = &MySql{Db: conn}
	dbObj.Crt_tb()

	stmt, err := dbObj.Db.Prepare("INSERT INTO project(ProjectName, ProjectType, FinishMark, FinishStr, DefaultEnv, LastModule, Mainfinished, SubmitEnabled, Start_time, Status, MaxRetriedTimes) values(?,?,?,?,?,?,?,?,?,?,?)")
	DAG2yaml.CheckErr(err)

	rows, err := dbObj.Db.Query("select ProjectName from project where ProjectName = ?", ProjectObj.ProjectName)
	defer rows.Close()
	DAG2yaml.CheckErr(err)

	now := time.Now().Format("2006-01-02 15:04:05")
	if CheckCount(rows)==0 {
		_, err = stmt.Exec(ProjectObj.ProjectName, ProjectObj.ProjectType, ProjectObj.FinishMark, ProjectObj.FinishStr, ProjectObj.DefaultEnv, ProjectObj.LastModule, ProjectObj.Mainfinished, 1, now, P_unsubmit, ProjectObj.MaxRetriedTimes)
		DAG2yaml.CheckErr(err)
	}

	update_stmt, err := dbObj.Db.Prepare("update project set Total=?, Unsubmit=?, Pending=?, Running=?, Failed=?, Succeeded=? where ProjectName=?")
	DAG2yaml.CheckErr(err)
	_, err = update_stmt.Exec(0,0,0,0,0,0,ProjectObj.ProjectName)
	DAG2yaml.CheckErr(err)

	cff = Taskconf.Config_Init()
	cff.AddPrj(ProjectObj.ProjectName, dbpath)
	cff.Update()

	return
}
*/
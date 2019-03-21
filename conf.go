package Taskconf

import (
	//	"fmt"
	"github.com/ahworld07/DAG2yaml"
	"github.com/go-ini/ini"
	"os"
	"path"
)

//This struct is used to read/write the config file, including default parameters and project database information.
type ConfigFile struct {
	Conffile    string
	Cfg         *ini.File
}

func (cff *ConfigFile)SetDefault(){
	Hname, err := os.Hostname()
	DAG2yaml.CheckErr(err)

	_ ,_ = cff.Cfg.NewSection("project")
	_ ,_ = cff.Cfg.NewSection("base")
	_ ,_ = cff.Cfg.NewSection("kubectl")
	_, err = cff.Cfg.Section("base").NewKey("CronNode",Hname)
	DAG2yaml.CheckErr(err)
	_, err = cff.Cfg.Section("base").NewKey("defaultFinishMark","Still_waters_run_deep")
	DAG2yaml.CheckErr(err)
	_, err = cff.Cfg.Section("base").NewKey("pobMaxRetries","3")
	DAG2yaml.CheckErr(err)
	_, err = cff.Cfg.Section("kubectl").NewKey("RunAsGroup","511")
	DAG2yaml.CheckErr(err)
	cff.Update()
}


func Config_Init()(cff *ConfigFile){
	home, _ := DAG2yaml.Home()
	Conf_file := path.Join(home, "gomonitor.project.conf")
	exit_file, err := DAG2yaml.PathExists(Conf_file)
	DAG2yaml.CheckErr(err)
	needupdate := false
	if exit_file == false {
		f, _ := os.OpenFile(Conf_file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
		defer f.Close()
		needupdate = true
	}
	cfg, err := ini.LoadSources(ini.LoadOptions{AllowBooleanKeys: true}, Conf_file)
	DAG2yaml.CheckErr(err)

	pobMaxRetries := cfg.Section("base").Key("pobMaxRetries").String()
	RunAsGroup := cfg.Section("kubectl").Key("RunAsGroup").String()

	if pobMaxRetries == "" || RunAsGroup == ""{
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
	DAG2yaml.CheckErr(err)
}

func (cff *ConfigFile)Update(){
	err := cff.Cfg.SaveTo(cff.Conffile)
	DAG2yaml.CheckErr(err)
}

func (cff *ConfigFile)RemovePrj(prjname string){
	prjdb := cff.Cfg.Section("project").Key(prjname).String()
	exit_file, err := DAG2yaml.PathExists(prjdb)
	DAG2yaml.CheckErr(err)
	if exit_file == true {
		err = os.Remove(prjdb)
		DAG2yaml.CheckErr(err)
	}
	cff.Cfg.Section("project").DeleteKey(prjname)
}

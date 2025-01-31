package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/mtesauro/godojo/config"
)

// Commands to bootstrap Ubuntu for the installer
func ubuntuInitOSInst(id string, b *osCmds) {
	switch strings.ToLower(id) {
	case "debian:10":
		fallthrough
	case "ubuntu:18.04":
		fallthrough
	case "ubuntu:20.04":
		fallthrough
	case "ubuntu:20.10":
		fallthrough
	case "ubuntu:21.04":
		b.id = id
		b.cmds = []string{
			fmt.Sprintf("curl -sS %s | apt-key add -", YarnGPG),
			fmt.Sprintf("echo -n %s > /etc/apt/sources.list.d/yarn.list", YarnRepo),
			"DEBIAN_FRONTEND=noninteractive apt-get update",
			"DEBIAN_FRONTEND=noninteractive apt-get install sudo",
			fmt.Sprintf("curl -sL %s | bash - ", NodeURL),
			"DEBIAN_FRONTEND=noninteractive apt-get install -y apt-transport-https libjpeg-dev gcc libssl-dev python3-dev python3-pip python3-virtualenv yarn build-essential expect",
		}
		b.errmsg = []string{
			"Unable to obtain the gpg key for Yarn",
			"Unable to add yard repo as an apt source",
			"Unable to update apt database",
			"Unable to install sudo",
			"Unable to install nodejs",
			"Installing OS packages with apt failed",
		}
		b.hard = []bool{
			true,
			true,
			true,
			true,
			true,
			true,
		}
		// Currently, only Ubuntu 18.04 is supported
	}
	return
}

// Commands to install SQLite on Ubuntu
func ubuntuInstSQLite(id string, b *osCmds) {
	switch id {
	case "ubuntu:18.04":
		fallthrough
	case "ubuntu:20.04":
		fallthrough
	case "ubuntu:20.10":
		fallthrough
	case "ubuntu:21.04":
		b.id = id
		b.cmds = []string{
			"DEBIAN_FRONTEND=noninteractive apt-get install -y sqlite3",
		}
		b.errmsg = []string{
			"Unable to install SQLite",
		}
		b.hard = []bool{
			true,
		}
	}
	return
}

// Commands to install MariaDB on Ubuntu
func ubuntuInstMariaDB(id string, b *osCmds) {
	switch id {
	case "ubuntu:18.04":
		fallthrough
	case "ubuntu:20.04":
		fallthrough
	case "ubuntu:20.10":
		fallthrough
	case "ubuntu:21.04":
		b.id = id
		b.cmds = []string{
			"DEBIAN_FRONTEND=noninteractive apt-get install -y mariadb-server libmariadbclient-dev",
		}
		b.errmsg = []string{
			"Unable to install MariaDB",
		}
		b.hard = []bool{
			true,
		}
	}
	return
}

// Commands to install MySQL on Ubuntu
func ubuntuInstMySQL(id string, b *osCmds) {
	traceMsg(fmt.Sprintf("Installing Ubuntu MySQL for %s\n", id))
	switch id {
	case "ubuntu:18.04":
		fallthrough
	case "ubuntu:20.04":
		fallthrough
	case "ubuntu:20.10":
		fallthrough
	case "ubuntu:21.04":
		b.id = id
		b.cmds = []string{
			"DEBIAN_FRONTEND=noninteractive apt-get install -y mysql-server libmysqlclient-dev",
		}
		b.errmsg = []string{
			"Unable to install MySQL",
		}
		b.hard = []bool{
			true,
		}
	}
	return
}

// Commands to install PostgreSQL on Ubuntu
func ubuntuInstPostgreSQL(id string, b *osCmds) {
	switch strings.ToLower(id) {
	case "debian:10":
		fallthrough
	case "ubuntu:18.04":
		fallthrough
	case "ubuntu:20.04":
		fallthrough
	case "ubuntu:20.10":
		fallthrough
	case "ubuntu:21.04":
		b.id = id
		b.cmds = []string{
			"DEBIAN_FRONTEND=noninteractive apt-get install -y libpq-dev postgresql postgresql-contrib postgresql-client-common",
		}
		b.errmsg = []string{
			"Unable to install PostgreSQL",
		}
		b.hard = []bool{
			true,
		}
	}
	return
}

func ubuntuInstPostgreSQLClient(id string, b *osCmds) {
	switch id {
	case "ubuntu:18.04":
		fallthrough
	case "ubuntu:20.04":
		fallthrough
	case "ubuntu:20.10":
		fallthrough
	case "ubuntu:21.04":
		b.id = id
		b.cmds = []string{
			"DEBIAN_FRONTEND=noninteractive apt-get install -y postgresql-client-12",
			"/usr/sbin/groupadd -f postgres",                         // TODO: consider using os.Group.Lookup before calling this
			"/usr/sbin/useradd -s /bin/bash -m -g postgres postgres", // TODO: consider using os.User.Lookup before calling this
		}
		b.errmsg = []string{
			"Unable to install PostgreSQL client",
			"Unable to add postgres group",
			"Unable to add postgres user",
		}
		b.hard = []bool{
			true,
			true,
			true,
		}
	}
	return
}

// Determine the default creds for a database freshly installed in Ubuntu
func ubuntuDefaultDBCreds(db *config.DBTarget, creds map[string]string) {
	// Installer currently assumes the default DB passwrod handling won't change by release
	// Switch on the DB type
	switch db.Engine {
	case "MySQL":
		ubuntuDefaultMySQL(creds)
	case "PostgreSQL":
		// Set creds as the Ruser & Rpass for Postgres
		creds["user"] = db.Ruser
		creds["pass"] = db.Rpass
		ubuntuDefaultPgSQL(creds)
	}

	return
}

func ubuntuDefaultMySQL(c map[string]string) {
	// Sent some initial values that ensure the connection will fail if the file read fails
	c["user"] = "debian-sys-maint"
	c["pass"] = "FAIL"

	// Pull the debian-sys-maint creds from /etc/mysql/debian.cnf
	f, err := os.Open("/etc/mysql/debian.cnf")
	if err != nil {
		// Exit with error code if we can't read the default creds file
		errorMsg("Unable to read file with defautl credentials, cannot continue")
		os.Exit(1)
	}

	// Create a new buffered reader
	fr := bufio.NewReader(f)

	// Create a scanner to run through the lines of the file
	scanner := bufio.NewScanner(fr)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "password") {
			l := strings.Split(line, "=")
			// Make sure there was a "=" in l
			if len(l) > 1 {
				c["pass"] = strings.Trim(l[1], " ")
				break
			}
		}
	}
	if err = scanner.Err(); err != nil {
		// Exit with error code if we can't scan the default creds file
		errorMsg("Unable to scan file with defautl credentials, cannot continue")
		os.Exit(1)
	}

}

func ubuntuDefaultPgSQL(creds map[string]string) {
	traceMsg("Called ubuntuDefaultPgSQL")

	// Sent user to postgres as that's the default DB user for any new install
	creds["user"] = "postgres"

	// Use the default local OS user to set the postgres DB user
	pgAlter := osCmds{
		id:     "linux",
		cmds:   []string{"sudo -u postgres psql -c \"ALTER USER postgres with encrypted password '" + creds["pass"] + "';\""},
		errmsg: []string{"Unable to set initial password for PostgreSQL database user postgres"},
		hard:   []bool{false},
	}

	// Try command
	err := tryCmds(cmdLogger, pgAlter)
	if err != nil {
		traceMsg(fmt.Sprintf("Error updating PostgreSQL DB user with %+v", squishSlice(pgAlter.cmds)))
		errorMsg("Unable to update default PostgreSQL DB user, quitting")
		os.Exit(1)
	}

	traceMsg("No error return from ubuntuDefaultPgSQL")
	return
}

func ubuntuOSPrep(id string, inst *config.InstallConfig, b *osCmds) {
	// Setup virutalenv, setup OS User, and chown DefectDojo app root to the dojo user
	switch id {
	case "ubuntu:18.04":
		fallthrough
	case "ubuntu:20.04":
		fallthrough
	case "ubuntu:20.10":
		fallthrough
	case "ubuntu:21.04":
		b.id = id
		b.cmds = []string{
			"python3 -m virtualenv --python=/usr/bin/python3 " + inst.Root,
			inst.Root + "/bin/python3 -m pip install --upgrade pip",
			inst.Root + "/bin/pip3 install -r " + inst.Root + "/django-DefectDojo/requirements.txt",
			"mkdir " + inst.Root + "/logs",
			"/usr/sbin/groupadd -f " + inst.OS.Group, // TODO: check with os.Group.Lookup
			"id " + inst.OS.User + " &>/dev/null; if [ $? -ne 0 ]; then useradd -s /bin/bash -m -g " + inst.OS.Group + " " + inst.OS.User + "; fi", // TODO: check with os.User.Lookup
			"chown -R " + inst.OS.User + "." + inst.OS.Group + " " + inst.Root,
		}
		b.errmsg = []string{
			"Unable to setup virtualenv for DefectDojo",
			"Unable to update pip to latest",
			"Unable to install Python3 modules for DefectDojo",
			"Unable to create a directory for logs",
			"Unable to create a group for DefectDojo OS user",
			"Unable to create an OS user for DefectDojo",
			"Unable to change ownership of the DefectDojo app root directory",
		}
		b.hard = []bool{
			true,
			true,
			true,
			true,
			true,
			true,
			true,
		}
	}

	return
}

func ubuntuSetupDDjango(id string, inst *config.InstallConfig, b *osCmds) {
	// Setup expect script needed to set initial admin password
	traceMsg(fmt.Sprintf("Injecting file %s at %s", "setup-superuser.expect", inst.Root+"/django-DefectDojo"))
	_ = injectFile("setup-superuser.expect", inst.Root+"/django-DefectDojo", 0755)

	err := patchOMatic(inst)
	if err != nil {
		traceMsg(fmt.Sprintf("patchOMatic failed with non-blocking error: %+v", err))
		traceMsg("A failure of patchOMatic may lead to a corrupt install - be warned")
	}

	// Django installs - migrations, create Django superuser
	// TODO: Remove this switch to simplify
	switch id {
	case "ubuntu:18.04":
		fallthrough
	case "ubuntu:20.04":
		fallthrough
	case "ubuntu:20.10":
		fallthrough
	case "ubuntu:21.04":
		// Add commands to setup DefectDojo - migrations, super user,
		// removed - "cd " + inst.Root + "/django-DefectDojo && source ../bin/activate && python3 manage.py makemigrations --merge --noinput", "Initial makemgrations failed",
		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py makemigrations dojo",
			"Failed during makemgration dojo", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py migrate",
			"Failed during database migrate", true)

		// Ensure there's a value for email as the call will fail without one
		adminEmail := "default.user@defectdojo.org"
		if len(inst.Admin.Email) > 0 {
			// If user configures an incorrect email, this will still fail but that's on them, not godojo
			adminEmail = inst.Admin.Email
		}
		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py createsuperuser --noinput --username=\""+
			inst.Admin.User+"\" --email=\""+adminEmail+"\"",
			"Failed while creating DefectDojo superuser", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && "+
			inst.Root+"/django-DefectDojo/setup-superuser.expect "+inst.Admin.User+" \""+escSpCar(inst.Admin.Pass)+"\"",
			"Failed while setting the password for the DefectDojo superuser", true)

		// Roles showed up in 2.x.x
		if onlyAfter(inst.Version, 2, 0, 0) {
			addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py loaddata role",
				"Failed while the loading data for role", true)
		}

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py loaddata product_type",
			"Failed while the loading data for product_type", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py loaddata test_type",
			"Failed while the loading data for test_type", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py loaddata development_environment",
			"Failed while the loading data for development_environment", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py loaddata system_settings",
			"Failed while the loading data for system_settings", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py loaddata benchmark_type",
			"Failed while the loading data for benchmark_type", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py loaddata benchmark_category",
			"Failed while the loading data for benchmark_category", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py loaddata benchmark_requirement",
			"Failed while the loading data for benchmark_requirement", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py loaddata language_type",
			"Failed while the loading data for language_type", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py loaddata objects_review",
			"Failed while the loading data for objects_review", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py loaddata regulation",
			"Failed while the loading data for regulation", true)

		// removed - "cd " + inst.Root + "/django-DefectDojo && source ../bin/activate && python3 manage.py import_surveys", "Failed while the running import_surveys",
		// removed - "cd " + inst.Root + "/django-DefectDojo && source ../bin/activate && python3 manage.py loaddata initial_surveys", "Failed while the loading data for initial_surveys",

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py buildwatson",
			"Failed while the running buildwatson", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo && source ../bin/activate && python3 manage.py installwatson",
			"Failed while the running installwatson", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo/components && yarn",
			"Failed while the running yarn", true)

		addCmd(b, "cd "+inst.Root+"/django-DefectDojo/ && source ../bin/activate && python3 manage.py collectstatic --noinput",
			"Failed while the running collectstatic", true)

		addCmd(b, "chown -R "+inst.OS.User+"."+inst.OS.Group+" "+inst.Root,
			"Unable to change ownership of the DefectDojo directory", true)
	}

	return
}

func injectFile(n string, p string, mask fs.FileMode) error {
	loc := emdir + n
	d, err := Asset(loc)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(p+"/"+n, d, mask)
	if err != nil {
		// File can't be written
		return err
	}

	traceMsg(fmt.Sprintf("Wrote file %s at %s", n, p))

	return nil
}

func patchOMatic(inst *config.InstallConfig) error {
	// If a source or commit install, do no patching
	if inst.SourceInstall {
		return nil
	}

	// Check the install version for any needed patches
	switch inst.Version {
	case "1.15.1":
		// Replace dojo/tools/factory to work around bug in Python 3.8 - https://bugs.python.org/issue44061
		w := bufio.NewWriter(os.Stdout)
		_ = injectFile("factory_2.0.3", inst.Root+"/django-DefectDojo/dojo/tools", 755)
		_ = tryCmd(w,
			"mv -f "+inst.Root+"/django-DefectDojo/dojo/tools/factory.py "+inst.Root+"/django-DefectDojo/dojo/tools/factory_py.buggy",
			"Error renaming factory.py to factory_py.buggy", false)
		_ = tryCmd(w,
			"mv -f "+inst.Root+"/django-DefectDojo/dojo/tools/factory_2.0.3 "+inst.Root+"/django-DefectDojo/dojo/tools/factory.py",
			"Error replacing factory.py with updated one from version 2.0.3", false)
	}

	return nil
}

//onlyAfter(inst.Version, "2")
func onlyAfter(v string, major int, minor int, patch int) bool {
	// Split up version
	vBits := strings.Split(v, ".")
	if len(vBits) != 3 {
		traceMsg(fmt.Sprintf("Bad version string: %s sent to onlyAfter()", v))
		return false
	}

	// Convert version bits
	vMaj, _ := strconv.Atoi(vBits[0])
	vMin, _ := strconv.Atoi(vBits[1])
	vPat, _ := strconv.Atoi(vBits[2])

	//
	if vMaj < major {
		return false
	}
	if vMin < minor {
		return false
	}
	if vPat < patch {
		return false
	}

	return true
}

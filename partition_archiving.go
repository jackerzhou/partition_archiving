package main

import (
	"database/sql"
	"flag"
	"fmt"
	gexpect "github.com/ThomasRooney/gexpect"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/smtp"
	"regexp"
	"strconv"
)

type hostConfig struct {
	dbHost       string
	dbUser       string
	dbPass       string
	dbName       string
	sshUser      string
	sshPass      string
	dataDir      string
	dbTable      string
	dbBackupName string
}

type smtpConfig struct {
	server    string
	port      int
	sender    string
	user      string
	password  string
	recipient string
}

type archiveStruct struct {
	source        hostConfig
	destination   hostConfig
	stepNum       int
	tmpTable      string
	fromStep      int
	partition     string
	smtpAlert     smtpConfig
	sourceDb      *sql.DB
	destinationDb *sql.DB
	err           error
	backupHost    string
	backupSshUser string
	backupSshPass string
	backupPath    string
	lastCommand   string
}

func main() {

	// Definitions

	archive := new(archiveStruct)

	archive.stepNum = 0

	// Read arguments

	flag.StringVar(&archive.source.dbHost, "source-db-host", "localhost", "Source DB Host")
	flag.StringVar(&archive.source.dbUser, "source-db-user", "root", "Source DB User")
	flag.StringVar(&archive.source.dbPass, "source-db-pass", "pass", "Source DB Pass")
	flag.StringVar(&archive.source.dbName, "source-db-name", "ISP", "Source DB Name")
	flag.StringVar(&archive.source.sshUser, "source-ssh-user", "root", "Source SSH User")
	flag.StringVar(&archive.source.sshPass, "source-ssh-pass", "pass", "Source SSH Pass")

	flag.StringVar(&archive.source.dataDir, "source-datadir", "/export/mysql/data", "Source MySQL Datadir")

	flag.StringVar(&archive.source.dbTable, "source-db-table", "RAD_ACCT", "Source DB Table")

	flag.StringVar(&archive.source.dbBackupName, "source-db-backup-name", "ISP_BKP", "Source DB backup name")

	flag.StringVar(&archive.destination.dbHost, "destination-db-host", "localhost", "Destination DB Host")
	flag.StringVar(&archive.destination.dbUser, "destination-db-user", "root", "Destination DB User")
	flag.StringVar(&archive.destination.dbPass, "destination-db-pass", "pass", "Destination DB Pass")
	flag.StringVar(&archive.destination.dbName, "destination-db-name", "ISP", "Destination DB Name")
	flag.StringVar(&archive.destination.sshUser, "destination-ssh-user", "root", "Destination SSH User")
	flag.StringVar(&archive.destination.sshPass, "destination-ssh-pass", "pass", "Destination SSH Pass")

	flag.StringVar(&archive.destination.dataDir, "destination-datadir", "/export/mysql/data", "Destination MySQL Datadir")

	flag.StringVar(&archive.destination.dbTable, "destination-db-table", "RAD_ACCT", "Destination DB Table")

	flag.StringVar(&archive.destination.dbBackupName, "destination-db-backup-name", "ISP_BKP", "Destination DB backup name")

	flag.StringVar(&archive.tmpTable, "tmp-table", "RAD_ACCT_TMP", "Temp table name")

	flag.IntVar(&archive.fromStep, "from-step", 1, "From step number")

	flag.StringVar(&archive.partition, "partition", "p180001", "Partition name to archive")

	flag.StringVar(&archive.smtpAlert.server, "smtp-server", "spamwall.netlabs.com.ar", "SMTP Server")
	flag.IntVar(&archive.smtpAlert.port, "smtp-port", 25, "SMTP server port")
	flag.StringVar(&archive.smtpAlert.sender, "smtp-sender", "alertas@netlabs.com.ar", "SMTP sender")
	flag.StringVar(&archive.smtpAlert.user, "smtp-user", "alertas@netlabs.com.ar", "SMTP user")
	flag.StringVar(&archive.smtpAlert.password, "smtp-password", "pass", "SMTP password")
	flag.StringVar(&archive.smtpAlert.recipient, "smtp-recipient", "alertas@netlabs.com.ar", "SMTP recipient")

	flag.StringVar(&archive.backupHost, "backup-host", "localhost", "Backup Host")
	flag.StringVar(&archive.backupSshUser, "backup-ssh-user", "root", "Backup SSH User")
	flag.StringVar(&archive.backupSshPass, "backup-ssh-pass", "pass", "Backup SSH Pass")
	flag.StringVar(&archive.backupPath, "backup-path", "/export/db/RAD_ACCT", "Backup Path")

	flag.Parse()

	// Add partition name to temporal table

	archive.tmpTable += "_" + archive.partition

	// Connect to DB Source

	archive.sourceDb, archive.err = sql.Open("mysql", archive.source.dbUser+":"+archive.source.dbPass+"@tcp("+archive.source.dbHost+":3306)/"+archive.source.dbName+"?charset=utf8")
	archive.checkErr("")

	// Connect to DB Destination

	archive.destinationDb, archive.err = sql.Open("mysql", archive.destination.dbUser+":"+archive.destination.dbPass+"@tcp("+archive.destination.dbHost+":3306)/"+archive.destination.dbName+"?charset=utf8")
	archive.checkErr("")

	// Create temp table in source db

	archive.runSQL("@"+archive.source.dbHost+": ", archive.sourceDb, "create database "+archive.source.dbBackupName)

	// Creates the temporal table in source host as from source db, without partitions

	var createTmpTable string
	archive.getCreateTable(archive.sourceDb, archive.source.dbName, archive.source.dbTable, archive.source.dbBackupName, archive.tmpTable, &createTmpTable)
	archive.runSQL("@"+archive.source.dbHost+": ", archive.sourceDb, createTmpTable)

	// Extraigo la particion que quiero archivar

	archive.runSQL("@"+archive.source.dbHost+": ", archive.sourceDb, "alter table "+archive.source.dbName+"."+archive.source.dbTable+" exchange partition "+archive.partition+" with table "+archive.source.dbBackupName+"."+archive.tmpTable)

	// Creo la base temporal en destino

	archive.runSQL("@"+archive.destination.dbHost+": ", archive.destinationDb, "create database "+archive.destination.dbBackupName)

	// Crea la tabla temporal igual que la destinio

	archive.getCreateTable(archive.destinationDb, archive.destination.dbName, archive.destination.dbTable, archive.destination.dbBackupName, archive.tmpTable, &createTmpTable)
	archive.runSQL("@"+archive.destination.dbHost+": ", archive.destinationDb, createTmpTable)

	// Hago el flush table with read lock para que me deje los archivos listos para copiar

	archive.runSQL("@"+archive.source.dbHost+": ", archive.sourceDb, "FLUSH TABLES "+archive.source.dbBackupName+"."+archive.tmpTable+" WITH READ LOCK")

	// Copio los files de la tabla

	archive.runSshCmd("/usr/bin/scp "+archive.source.sshUser+"@"+archive.source.dbHost+":"+archive.source.dataDir+"/"+archive.source.dbBackupName+"/"+archive.tmpTable+".* /tmp", archive.source.sshPass)

	// Desmonto el tablespace de la tabla destino

	archive.runSQL("@"+archive.destination.dbHost+": ", archive.destinationDb, "alter table "+archive.destination.dbBackupName+"."+archive.tmpTable+" discard tablespace")

	// Borro los files de la tabla temporal en destino

	archive.runSshCmd("/usr/bin/ssh "+archive.destination.sshUser+"@"+archive.destination.dbHost+" rm "+archive.destination.dataDir+"/"+archive.destination.dbBackupName+"/"+archive.tmpTable+".*", archive.destination.sshPass)

	// Copio los files de la tabla a destino

	archive.runSshCmd("/usr/bin/scp /tmp/"+archive.tmpTable+".cfg "+"/tmp/"+archive.tmpTable+".frm /tmp/"+archive.tmpTable+".ibd "+archive.destination.sshUser+"@"+archive.destination.dbHost+":"+archive.destination.dataDir+"/"+archive.destination.dbBackupName, archive.destination.sshPass)

	// Cambio el ownership de los archivos copiados

	archive.runSshCmd("/usr/bin/ssh "+archive.destination.sshUser+"@"+archive.destination.dbHost+" chown mysql:mysql "+archive.destination.dataDir+"/"+archive.destination.dbBackupName+"/"+archive.tmpTable+".*", archive.destination.sshPass)

	// Monto el tablespace con los files copiados

	archive.runSQL("@"+archive.destination.dbHost+": ", archive.destinationDb, "alter table "+archive.destination.dbBackupName+"."+archive.tmpTable+" import tablespace")

	// Reinserto la partiticion en la tabla de backup

	archive.runSQL("@"+archive.destination.dbHost+": ", archive.destinationDb, "alter table "+archive.destination.dbName+"."+archive.destination.dbTable+" exchange partition "+archive.partition+" with table "+archive.destination.dbBackupName+"."+archive.tmpTable)

	// Me desconecto de las bases

	archive.sourceDb.Close()
	archive.destinationDb.Close()

	// Comienzo procedimiento de purga de las tablas temporales de origen y destino

	// Conecto a la DB Source

	archive.sourceDb, archive.err = sql.Open("mysql", archive.source.dbUser+":"+archive.source.dbPass+"@tcp("+archive.source.dbHost+":3306)/"+archive.source.dbName+"?charset=utf8")
	archive.checkErr("")

	// Conecto a la DB Destination

	archive.destinationDb, archive.err = sql.Open("mysql", archive.destination.dbUser+":"+archive.destination.dbPass+"@tcp("+archive.destination.dbHost+":3306)/"+archive.destination.dbName+"?charset=utf8")
	archive.checkErr("")

	// Borro la tabla temporal de destino

	archive.runSQL("@"+archive.destination.dbHost+": ", archive.destinationDb, "drop table "+archive.destination.dbBackupName+"."+archive.tmpTable)

	// Borro la base temporal

	archive.runSQL("@"+archive.destination.dbHost+": ", archive.destinationDb, "drop database "+archive.destination.dbBackupName)

	// Borro la tabla temporal de origen

	archive.runSQL("@"+archive.source.dbHost+": ", archive.sourceDb, "drop table "+archive.source.dbBackupName+"."+archive.tmpTable)

	// Borro la base temporal de origen

	archive.runSQL("@"+archive.source.dbHost+": ", archive.sourceDb, "drop database "+archive.source.dbBackupName)

	// Comprimo los archivos del tablespace

	archive.runLocalCmd("tar czvvf " + archive.tmpTable + ".tgz " + archive.tmpTable + ".cfg " + archive.tmpTable + ".frm " + archive.tmpTable + ".ibd")

	// Salgo si el archivo existe en el servidor de backup

	archive.runSshCmd("/usr/bin/ssh "+archive.backupSshUser+"@"+archive.backupHost+" if [ -f "+archive.backupPath+"/"+archive.tmpTable+".tgz ] ; then echo \"file "+archive.backupPath+"/"+archive.tmpTable+".tgz already exists!!!!\"; exit 1; fi", archive.backupSshPass)

	// Copio los files al servidor de backup

	archive.runSshCmd("/usr/bin/scp /tmp/"+archive.tmpTable+".tgz "+archive.backupSshUser+"@"+archive.backupHost+":"+archive.backupPath+"/", archive.backupSshPass)

	// Hago los files read-only

	archive.runSshCmd("/usr/bin/ssh "+archive.backupSshUser+"@"+archive.backupHost+" chmod 400 "+archive.backupPath+"/"+archive.tmpTable+".tgz", archive.backupSshPass)

	// Me desconecto de las bases

	archive.sourceDb.Close()
	archive.destinationDb.Close()

}

func (archive *archiveStruct) checkErr(msg string) {
	if archive.err != nil {
		if msg != "" {
			msg = archive.lastCommand + "\n" + msg + "\n\n" + archive.err.Error()
		} else {
			msg = archive.lastCommand + "\n" + archive.err.Error()
		}

		archive.sendMail("Problema en el paso "+strconv.Itoa(archive.stepNum), msg)
		panic(archive.err)
	}
}

func (archive *archiveStruct) printBanner(msg string) {
	archive.lastCommand = msg

	fmt.Println("***************")
	fmt.Println("* Step " + strconv.Itoa(archive.stepNum))
	fmt.Println("***************")
	fmt.Println()
	fmt.Println(msg)
	fmt.Println()

}

func (archive *archiveStruct) getCreateTable(db *sql.DB, dbName string, dbTable string, dbBackupName string, tmpTable string, sqlQuery *string) {
	var rows *sql.Rows

	cmd := "show create table " + dbName + "." + dbTable

	archive.stepNum++

	if archive.stepNum < archive.fromStep {
		archive.printBanner("Skipping: " + cmd)
		return
	}

	archive.printBanner(cmd)

	rows, archive.err = db.Query(cmd)
	archive.checkErr("")

	for rows.Next() {
		var table string
		var createTable string

		archive.err = rows.Scan(&table, &createTable)
		archive.checkErr("")

		re := regexp.MustCompile("(?s)/\\*.*")
		*sqlQuery = re.ReplaceAllString(createTable, "")

		re = regexp.MustCompile("AUTO_INCREMENT=\\d+")
		*sqlQuery = re.ReplaceAllString(*sqlQuery, "")

		re = regexp.MustCompile("CREATE TABLE `" + dbTable + "`")
		*sqlQuery = re.ReplaceAllString(*sqlQuery, "CREATE TABLE `"+dbBackupName+"`.`"+tmpTable+"`")
	}
}

func (archive *archiveStruct) runSQL(msg string, db *sql.DB, sql string) {

	archive.stepNum++

	if archive.stepNum < archive.fromStep {
		archive.printBanner("Skipping: " + msg + sql)
		return
	}

	archive.printBanner(msg + sql)

	_, archive.err = db.Query(sql)

	archive.checkErr("")
}

func (archive *archiveStruct) runSshCmd(cmd string, password string) {

	archive.stepNum++

	if archive.stepNum < archive.fromStep {
		archive.printBanner("Skipping: " + cmd)
		return
	}

	archive.printBanner(cmd)

	var child *gexpect.ExpectSubprocess

	child, archive.err = gexpect.SpawnAtDirectory(cmd, "/tmp")
	archive.checkErr("")

	child.Expect("assword:")
	child.SendLine(password)

	var buff, output string
	output = ""

	for buff, archive.err = child.ReadLine(); buff != ""; buff, archive.err = child.ReadLine() {
		archive.checkErr("")
		output += buff

		fmt.Println(buff)
	}

	archive.err = child.Wait()
	archive.checkErr(output)

	child.Close()
}

func (archive *archiveStruct) runLocalCmd(cmd string) {

	archive.stepNum++

	if archive.stepNum < archive.fromStep {
		archive.printBanner("Skipping: " + cmd)
		return
	}

	archive.printBanner(cmd)

	var child *gexpect.ExpectSubprocess

	child, archive.err = gexpect.SpawnAtDirectory(cmd, "/tmp")
	archive.checkErr("")

	child.Interact()
	child.Wait()
	child.Close()
}

func (archive *archiveStruct) sendMail(subject string, body string) {
	// Connect to the remote SMTP server.
	c, err := smtp.Dial(archive.smtpAlert.server + ":" + strconv.Itoa(archive.smtpAlert.port))
	if err != nil {
		log.Fatal(err)
	}

	// Set the sender and recipient first
	if err := c.Mail(archive.smtpAlert.sender); err != nil {
		log.Fatal(err)
	}
	if err := c.Rcpt(archive.smtpAlert.recipient); err != nil {
		log.Fatal(err)
	}

	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		log.Fatal(err)
	}

	_, err = fmt.Fprintf(wc, "From: "+archive.smtpAlert.sender+"\n")
	if err != nil {
		log.Fatal(err)
	}

	_, err = fmt.Fprintf(wc, "To: "+archive.smtpAlert.recipient+"\n")
	if err != nil {
		log.Fatal(err)
	}

	_, err = fmt.Fprintf(wc, "Subject: "+subject+"\n")
	if err != nil {
		log.Fatal(err)
	}

	_, err = fmt.Fprintf(wc, "\n")
	if err != nil {
		log.Fatal(err)
	}

	_, err = fmt.Fprintf(wc, body)
	if err != nil {
		log.Fatal(err)
	}
	err = wc.Close()
	if err != nil {
		log.Fatal(err)
	}

	// Send the QUIT command and close the connection.
	err = c.Quit()
	if err != nil {
		log.Fatal(err)
	}
}

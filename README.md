# partition_archiving
## Archiving for MySQL partitions

### partition_archiving is a tool to extract partitions from a MySQL partitioned table, move it to another database, and copy the files to a backup server.

```
 Usage of /go/bin/partition_archiving:
   -backup-host string
    	Backup Host (default "localhost")
  -backup-path string
    	Backup Path (default "/export/db/RAD_ACCT")
  -backup-ssh-pass string
    	Backup SSH Pass (default "pass")
  -backup-ssh-user string
    	Backup SSH User (default "root")
  -destination-datadir string
    	Destination MySQL Datadir (default "/export/mysql/data")
  -destination-db-backup-name string
    	Destination DB backup name (default "ISP_BKP")
  -destination-db-host string
    	Destination DB Host (default "localhost")
  -destination-db-name string
    	Destination DB Name (default "ISP")
  -destination-db-pass string
    	Destination DB Pass (default "pass")
  -destination-db-table string
    	Destination DB Table (default "RAD_ACCT")
  -destination-db-user string
    	Destination DB User (default "root")
  -destination-ssh-pass string
    	Destination SSH Pass (default "pass")
  -destination-ssh-user string
    	Destination SSH User (default "root")
  -from-step int
    	From step number (default 1)
  -partition string
    	Partition name to archive (default "p180001")
  -smtp-password string
    	SMTP password (default "pass")
  -smtp-port int
    	SMTP server port (default 25)
  -smtp-recipient string
    	SMTP recipient (default "alertas@netlabs.com.ar")
  -smtp-sender string
    	SMTP sender (default "alertas@netlabs.com.ar")
  -smtp-server string
    	SMTP Server (default "spamwall.netlabs.com.ar")
  -smtp-user string
    	SMTP user (default "alertas@netlabs.com.ar")
  -source-datadir string
    	Source MySQL Datadir (default "/export/mysql/data")
  -source-db-backup-name string
    	Source DB backup name (default "ISP_BKP")
  -source-db-host string
    	Source DB Host (default "localhost")
  -source-db-name string
    	Source DB Name (default "ISP")
  -source-db-pass string
    	Source DB Pass (default "pass")
  -source-db-table string
    	Source DB Table (default "RAD_ACCT")
  -source-db-user string
    	Source DB User (default "root")
  -source-ssh-pass string
    	Source SSH Pass (default "pass")
  -source-ssh-user string
    	Source SSH User (default "root")
  -tmp-table string
    	Temp table name (default "RAD_ACCT_TMP")
```

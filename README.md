# partition_archiving
## Archiving for MySQL partitions

### partition_archiving is a tool written in golang to extract partitions from a MySQL partitioned table, move it to another database, and copy the files to a backup server.

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

### This tool was inspired by this work https://www.percona.com/live/mysql-conference-2013/sites/default/files/slides/discard_inport_exchange.pdf

## Example:

```
run.sh:

/go/bin/partition_archiving \
        -source-db-host 192.168.0.232 \
        -source-db-pass pass \
        -source-ssh-pass pass \
        -source-db-backup-name ISP_BKP4 \
        -partition p201303 \
        -destination-db-host 192.168.0.156 \
        -destination-db-pass pass \
        -destination-db-backup-name ISP_BKP3 \
        -destination-db-name ISP_B \
        -destination-ssh-pass pass \
        -destination-datadir /export/db \
        -tmp-table RAD_ACCT_TMP2 \
        -smtp-password pass \
        -smtp-recipient diegow@netlabs.com.ar \
        -backup-host 192.168.0.156 \
        -backup-ssh-pass pass \
        $*

./run.sh -partition p201305

***************
* Step 1
***************

@192.168.0.232: select * from ISP.RAD_ACCT partition (p201305) limit 1

***************
* Step 2
***************

@192.168.0.232: create database ISP_BKP4

***************
* Step 3
***************

show create table ISP.RAD_ACCT

***************
* Step 4
***************

@192.168.0.232: CREATE TABLE `ISP_BKP4`.`RAD_ACCT_TMP2_p201305` (
  `RAD_ACCT_ID` bigint(20) NOT NULL AUTO_INCREMENT,
  `ACCTSESSIONID` varchar(255) DEFAULT NULL,
  `ACCTUNIQUEID` varchar(255) DEFAULT NULL,
  `USERNAME` varchar(255) DEFAULT NULL,
...
  PRIMARY KEY (`RAD_ACCT_ID`,`ACCTSTARTTIME`),
  KEY `IDX_ACCTSESSIONID` (`ACCTSESSIONID`),
) ENGINE=InnoDB  DEFAULT CHARSET=utf8


***************
* Step 5
***************

@192.168.0.232: alter table ISP.RAD_ACCT exchange partition p201305 with table ISP_BKP4.RAD_ACCT_TMP2_p201305

***************
* Step 6
***************

@192.168.0.156: create database ISP_BKP3

***************
* Step 7
***************

show create table ISP_B.RAD_ACCT

***************
* Step 8
***************

@192.168.0.156: CREATE TABLE `ISP_BKP3`.`RAD_ACCT_TMP2_p201305` (
  `RAD_ACCT_ID` bigint(20) NOT NULL AUTO_INCREMENT,
  `ACCTSESSIONID` varchar(255) DEFAULT NULL,
  `ACCTUNIQUEID` varchar(255) DEFAULT NULL,
  `USERNAME` varchar(255) DEFAULT NULL,
...
  PRIMARY KEY (`RAD_ACCT_ID`,`ACCTSTARTTIME`),
  KEY `IDX_ACCTSESSIONID` (`ACCTSESSIONID`),
) ENGINE=InnoDB  DEFAULT CHARSET=utf8


***************
* Step 9
***************

@192.168.0.232: FLUSH TABLES ISP_BKP4.RAD_ACCT_TMP2_p201305 WITH READ LOCK

***************
* Step 10
***************

/usr/bin/scp root@192.168.0.232:/export/mysql/data/ISP_BKP4/RAD_ACCT_TMP2_p201305.* /tmp


RAD_ACCT_TMP2_p201305.cfg                     100% 2504     2.5KB/s   00:00
RAD_ACCT_TMP2_p201305.frm                     100%   22KB  21.5KB/s   00:00
RAD_ACCT_TMP2_p201305.ibd                     100%   80MB  40.0MB/s   00:02
***************
* Step 11
***************

@192.168.0.156: alter table ISP_BKP3.RAD_ACCT_TMP2_p201305 discard tablespace

***************
* Step 12
***************

/usr/bin/ssh root@192.168.0.156 rm /export/db/ISP_BKP3/RAD_ACCT_TMP2_p201305.*


***************
* Step 13
***************

/usr/bin/scp /tmp/RAD_ACCT_TMP2_p201305.cfg /tmp/RAD_ACCT_TMP2_p201305.frm /tmp/RAD_ACCT_TMP2_p201305.ibd root@192.168.0.156:/export/db/ISP_BKP3


RAD_ACCT_TMP2_p201305.cfg                     100% 2504     2.5KB/s   00:00
RAD_ACCT_TMP2_p201305.frm                     100%   22KB  21.5KB/s   00:00
RAD_ACCT_TMP2_p201305.ibd                     100%   80MB  13.3MB/s   00:06
***************
* Step 14
***************

/usr/bin/ssh root@192.168.0.156 chown mysql:mysql /export/db/ISP_BKP3/RAD_ACCT_TMP2_p201305.*


***************
* Step 15
***************

@192.168.0.156: alter table ISP_BKP3.RAD_ACCT_TMP2_p201305 import tablespace

***************
* Step 16
***************

@192.168.0.156: alter table ISP_B.RAD_ACCT exchange partition p201305 with table ISP_BKP3.RAD_ACCT_TMP2_p201305

***************
* Step 17
***************

@192.168.0.156: drop table ISP_BKP3.RAD_ACCT_TMP2_p201305

***************
* Step 18
***************

@192.168.0.156: drop database ISP_BKP3

***************
* Step 19
***************

@192.168.0.232: drop table ISP_BKP4.RAD_ACCT_TMP2_p201305

***************
* Step 20
***************

@192.168.0.232: drop database ISP_BKP4

***************
* Step 21
***************

tar czvvf RAD_ACCT_TMP2_p201305.tgz RAD_ACCT_TMP2_p201305.cfg RAD_ACCT_TMP2_p201305.frm RAD_ACCT_TMP2_p201305.ibd

-rw-r----- root/root      2504 2017-06-22 15:25 RAD_ACCT_TMP2_p201305.cfg
-rw-r----- root/root     22060 2017-06-22 15:25 RAD_ACCT_TMP2_p201305.frm
-rw-r----- root/root  83886080 2017-06-22 15:25 RAD_ACCT_TMP2_p201305.ibd
***************
* Step 22
***************

/usr/bin/ssh root@192.168.0.156 if [ -f /export/db/RAD_ACCT/RAD_ACCT_TMP2_p201305.tgz ] ; then echo "file /export/db/RAD_ACCT/RAD_ACCT_TMP2_p201305.tgz already exists!!!!"; exit 1; fi


***************
* Step 23
***************

/usr/bin/scp /tmp/RAD_ACCT_TMP2_p201305.tgz root@192.168.0.156:/export/db/RAD_ACCT/


RAD_ACCT_TMP2_p201305.tgz                     100%   21MB   1.2MB/s   00:18
***************
* Step 24
***************

/usr/bin/ssh root@192.168.0.156 chmod 400 /export/db/RAD_ACCT/RAD_ACCT_TMP2_p201305.tgz



In order to build partition string like YearMonth, from 12 months ago:

date -d "today - 12 month" "+%Y%m"


```

### In order to build partition string like YearMonth, from 12 months ago:

date -d "today - 12 month" "+%Y%m"

So running the command would be like this:

./run.sh -partition p`date -d "today - 12 month" "+%Y%m"`

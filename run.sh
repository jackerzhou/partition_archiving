/go/bin/partition_archiving \
	-source-db-host 192.168.0.232 \
	-source-db-pass m1SQl03r \
	-source-ssh-pass netlabs123 \
	-source-db-backup-name ISP_BKP4 \
	-partition p201303 \
	-destination-db-host 192.168.0.156 \
	-destination-db-pass nlabs \
	-destination-db-backup-name ISP_BKP3 \
	-destination-db-name ISP_B \
	-destination-ssh-pass a \
	-destination-datadir /export/db \
	-tmp-table RAD_ACCT_TMP2 \
	-smtp-password 4l3rt4s \
	-smtp-recipient diegow@netlabs.com.ar \
	-backup-host 192.168.0.156 \
	-backup-ssh-pass a \
	$*

# Files Removal Enforced Daemon

FRD is a daemon which allow to remove files from your directories. If you use watcher_type = "event" FRD will delete files as soon as they are created and if they match ext_file and file_size. If you need to check a gap between the file date and system date you should use watcher_type="loop".  
FRD can send trace of removed files to a graylog server.  
WARNING : test FRD with a test directory  
WARNING2 : FRD has not been tested on Windows at this time, take care.


## Build

	git clone https://github.com/fredix/frd
	cd /home/user/sources/frd/src
	
	Linux :

	go build -o ../release/linux/frd frd.go

	windows :

	32 bits :

	GOOS=windows GOARCH=386 go build -o ../release/win/frd_i386.exe frd.go

	64 bits :

	GOOS=windows GOARCH=amd64 go build -o ../release/win/frd_amd64.exe frd.go

### create 32 bits windows service

	sc create frd binpath= "\"C:\Users\user\sources\frd\win\frd_i386.exe\" \"C:\Users\user\sources\frd\win\frd.toml\" --service" depend= Tcpip


## Setup

cp frd /usr/local/bin/  
edit frd.toml and copy to /etc


## systemd service file

/usr/lib/systemd/user/frd.service

	[Unit]
	Description=FRD
	
	[Service]
	Type=simple
	ExecStart=/usr/local/bin/frd /etc/frd.toml
	StandardOutput=null
	Restart=on-failure
	
	[Install]
	WantedBy=multi-user.target
	Alias=frd.service

start service

	systemctl --user enable frd
	systemctl --user start frd 
	systemctl --user status frd -l 

## System V

/etc/init.d/frd 

	#!/bin/bash
	#
	#	/etc/rc.d/init.d/frd
	#
	#	<description of the *service*>
	#	<any general comments about this init script>
	#
	# chkconfig: 345 70 30
	# Source function library.
	. /etc/init.d/functions

	start() {
		echo -n "Starting frd: "
		/usr/local/bin/frd /etc/frd.toml > /var/log/frd.log 2>&1 &
		touch /var/lock/subsys/frd
		return 0
	}	

	stop() {
		echo -n "Shutting down frd: "
		killall frd
		rm -f /var/lock/subsys/frd
		return 0
	}

	case "$1" in
	    start)
		start
		;;
	    stop)
		stop
		;;
	    status)
		;;
	    restart)
	    	stop
		start
		;;
	    reload)
		;;
	    condrestart)
		[ -f /var/lock/subsys/frd ] && restart
		;;
	    *)
		echo "Usage: frd {start|stop|status|reload|restart"
		exit 1
		;;
	esac
	exit $?


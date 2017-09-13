# Files Removal Enforced Daemon

FRED is a daemon which allow to remove files from your directories. If you use watcher_type = "event" FRED will delete files as soon as they are created and if they match ext_file and file_size. If you need to check a gap between the file date and system date you should use watcher_type="loop".  
FRED can send trace of removed files to a graylog server.  
WARNING : test FRED with a test directory  
WARNING2 : FRED has not been tested on Windows at this time, take care.


## Build

	export GOPATH=$GOPATH:/home/user/sources/fred
	
	cd /home/user/sources/fred/src
	
	Linux :

	go build -o ../release/linux/fred fred.go

	windows :

	32 bits :

	GOOS=windows GOARCH=386 go build -o ../release/win/fred_i386.exe fred.go

	64 bits :

	GOOS=windows GOARCH=amd64 go build -o ../release/win/fred_amd64.exe fred.go

### create 32 bits windows service

	sc create fred binpath= "\"C:\Users\user\sources\fred\win\fred_i386.exe\" \"C:\Users\user\sources\fred\win\fred.toml\" --service" depend= Tcpip


## Setup

cp fred /usr/local/bin/  
edit fred.toml and copy to /etc


## systemd service file

/usr/lib/systemd/user/fred.service

	[Unit]
	Description=FRED
	
	[Service]
	Type=simple
	ExecStart=/usr/local/bin/fred /etc/fred.toml
	StandardOutput=null
	Restart=on-failure
	
	[Install]
	WantedBy=multi-user.target
	Alias=fred.service

start service

	systemctl --user enable fred
	systemctl --user start fred 
	systemctl --user status fred -l 

## System V

/etc/init.d/fred 

	#!/bin/bash
	#
	#	/etc/rc.d/init.d/fred
	#
	#	<description of the *service*>
	#	<any general comments about this init script>
	#
	# chkconfig: 345 70 30
	# Source function library.
	. /etc/init.d/functions

	start() {
		echo -n "Starting fred: "
		/usr/local/bin/fred /etc/fred.toml > /var/log/fred.log 2>&1 &
		touch /var/lock/subsys/fred
		return 0
	}	

	stop() {
		echo -n "Shutting down fred: "
		killall fred
		rm -f /var/lock/subsys/fred
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
		[ -f /var/lock/subsys/fred ] && restart
		;;
	    *)
		echo "Usage: fred {start|stop|status|reload|restart"
		exit 1
		;;
	esac
	exit $?


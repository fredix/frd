# Files Removal Enforced Daemon

FRED est un daemon qui permet de supprimer des fichiers dans des répertoires. Il peut scruter les répertoires sur une boucle de temps définie (loop) ou en temps réel (event).
FRED peut envoyer une trace des fichiers supprimés vers un serveur Graylog. 


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

copier le binaire fred dans /usr/local/bin  
éditer le fichier fred.toml et le copier dans /etc


## Créer un fichier service 

	cat /usr/lib/systemd/system/fred.service


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


	systemctl daemon-reload
	systemctl enable fred
	systemctl start fred 
	systemctl status fred -l 

## Créer un fichier init.d pour les centos < 7

	cat /etc/init.d/fred 

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


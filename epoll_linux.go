package main

import (
	"log"
	"net"
	"os"
	"syscall"
)

func main() {
	var err error
	var lsn net.Listener
	var lsnFile *os.File
	var serfd int
	var epfd int
	var event syscall.EpollEvent
	var events []syscall.EpollEvent
	var eventsNum int
	var clifd int
	var cliAddr syscall.Sockaddr
	var addr *syscall.SockaddrInet4
	var buf = make([]byte, 10)

	// create server socket
	if lsn, err = net.Listen("tcp", "127.0.0.1:9999"); err != nil {
		log.Fatal(err)
	}
	if lsnFile, err = lsn.(*net.TCPListener).File(); err != nil {
		log.Fatal(err)
	}
	serfd = int(lsnFile.Fd())

	// epoll create
	if epfd, err = syscall.EpollCreate1(0); err != nil {
		log.Fatal(err)
	}
	defer syscall.Close(epfd)
	log.Println("epoll create success, epfd:", epfd)

	// epoll ctrl server fd
	event = syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(serfd)}
	if err = syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, serfd, &event); err != nil {
		log.Fatal(err)
	}

	// epoll wait
	events = make([]syscall.EpollEvent, 100)
	var i = 0
	var n = 0
	var evfd = 0
	for {
		if eventsNum, err = syscall.EpollWait(epfd, events, -1); err != nil {
			log.Fatal(err)
		}
		log.Println("epoll wait success, eventsNum:", eventsNum)

		for i = 0; i < eventsNum; i++ {
			evfd = int(events[i].Fd)
			// accept
			if evfd == serfd {
				if clifd, cliAddr, err = syscall.Accept(serfd); err != nil {
					log.Fatal(err)
				}
				addr = cliAddr.(*syscall.SockaddrInet4)
				log.Println("accept success, clifd:", clifd, "cliAddr:", addr.Addr, addr.Port)

				event = syscall.EpollEvent{Events: syscall.EPOLLIN, Fd: int32(clifd)}
				if err = syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, clifd, &event); err != nil {
					log.Fatal(err)
				}
			} else {
				// read
				if n, err = syscall.Read(evfd, buf); err != nil || n <= 0 {
					log.Println("cli closed:", evfd, "err:", err)
					event = syscall.EpollEvent{Fd: int32(evfd)}
					if err = syscall.EpollCtl(epfd, syscall.EPOLL_CTL_DEL, evfd, &event); err != nil {
						log.Fatal(err)
					}
					if err = syscall.Close(evfd); err != nil {
						log.Fatal(err)
					}
					continue
				}
				log.Println("read from cli:", evfd, string(buf[:n]))
			}

			// todo:根据event事件类型进行rw
		}
	}
}

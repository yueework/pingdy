package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/go-ping/ping"
)

var usage = `
Usage:
    ping [-c count] [-i interval] [-t timeout] [-z fail threshold] [-f full log] [--privileged] host
Examples:
	# logs when RTT (round trip time) exceeds 100ms with full log
	$ pingdy -f -i 200ms -z 100ms 192.168.99.20

	# logs when RTT exceeds 187ms for 100 packets
	$ pingdy -c 100 -i 400ms -z 187ms 192.168.99.20

	# logs when RTT exceeds 0.890ms aka 890microseconds
	pingdy -i 200ms -z 0.890ms 192.168.99.20

	# logs when RTT exceeds 2000ms aka 2seconds
	pingdy -i 200ms -z 1000ms 192.168.99.20

	# will try for 3s and quit if no reply
	pingdy -t 3s -i 500ms -z 187ms  192.168.99.20
`

func main() {
	timeout := flag.Duration("t", time.Second*100000, "")
	interval := flag.Duration("i", time.Second, "")
	count := flag.Int("c", -1, "")
	privileged := flag.Bool("privileged", false, "")
	threshold := flag.Duration("z", time.Millisecond, "")
	full := flag.Bool("f", false, "")
	flag.Usage = func() {
		fmt.Print(usage)
	}
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		return
	}

	host := flag.Arg(0)
	pinger, err := ping.NewPinger(host)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	}

	// listen for ctrl-C signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			pinger.Stop()
		}
	}()

	pinger.OnRecv = func(pkt *ping.Packet) {
		if pkt.Rtt > *threshold {
			fmt.Printf("%v ~~ %d bytes from %s: icmp_seq=%d rtt=%v ttl=%v\n",
				time.Now().Format("Mon Jan _2 15:04:05 PST 2006"), pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt, pkt.Ttl)
		}
		if *full {
			fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v ttl=%v\n",
				pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt, pkt.Ttl)
		}

	}
	pinger.OnDuplicateRecv = func(pkt *ping.Packet) {
		fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v ttl=%v (DUP!)\n",
			pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt, pkt.Ttl)
	}
	pinger.OnFinish = func(stats *ping.Statistics) {
		fmt.Printf("\n--- %s ping statistics ---\n", stats.Addr)
		fmt.Printf("%d packets transmitted, %d packets received, %d duplicates, %v%% packet loss\n",
			stats.PacketsSent, stats.PacketsRecv, stats.PacketsRecvDuplicates, stats.PacketLoss)
		fmt.Printf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
			stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
	}

	pinger.Count = *count
	pinger.Interval = *interval
	pinger.Timeout = *timeout
	pinger.SetPrivileged(*privileged)
	pinger.Size = 64
	pinger.Debug = true

	fmt.Printf("Pingdy -----> %s (%s):\n", pinger.Addr(), pinger.IPAddr())
	fmt.Printf("To log on threshold : %v\n", *threshold)
	err = pinger.Run()
	if err != nil {
		fmt.Printf("Failed to ping target host: %s", err)
	}

	stats := pinger.Statistics()
	fmt.Printf("%v", stats.MaxRtt)
}

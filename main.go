
package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"regexp"
)

type User struct {
	Nick string
	User string
	Host string
}

type Message struct {
	User User
	Channel string
	Content string
	Action bool
}

/**
 *	Prints usage information and exits
 */
func usage() {
	fmt.Fprintf( os.Stderr, "\033[1mUSAGE\033[0m: go run %s [server[:port]] [nick] [channel [channel...]]\n", os.Args[0] )
	flag.PrintDefaults()
	os.Exit( 2 )
}

/**
 *	Instantly dies with the given error message
 */
func die( message string ) {
	fmt.Fprintf( os.Stderr, "\033[31;1mERROR\033[0m: %s\n", message )
}

func main() {

	debug := true

	// Regexes
	privmsgRegexp := regexp.MustCompile( "^:(.+?)!(.+?)@(.+?)\\sPRIVMSG\\s(.+?)\\s:(.+)$" )
	modeRegexp    := regexp.MustCompile( "^:(.+?)!(.+?)@(.+?)\\sMODE\\s(.+?)\\s(.+)$" )
	
	// Get params
	if len( os.Args ) < 4 {
		usage()
	}
	
	host    := os.Args[1]
	nick    := os.Args[2]
	channels := os.Args[3:len(os.Args)]
	
	if host == "" || nick == "" {
		usage()
	}
	
	// Connect to the server
	addr, err := net.ResolveTCPAddr( "tcp", host )
	if err != nil {
		die( "Unable to resolve TCP Address " + host )
	}
	
	// Dial the socket
	socket, err := net.DialTCP( "tcp", nil, addr )
	if err != nil {
		die( "Unable to dial socket" )
	}
	
	// Create reader and writer objects for the socket
	reader := bufio.NewReader( socket )
	writer := bufio.NewWriter( socket )
	
	// Create channels for the reader and writer to communicate on
	read  := make( chan string, 1000 )
	write := make( chan string, 1000 )
	
	// Spin off tasks to handle reading and writing so that the bot doesn't
	// block when doing those things
	go func() {
		for {
			str, err := reader.ReadString( byte( '\n' ) )
			if err != nil {
				fmt.Fprintf( os.Stderr, "\033[31;1mERROR\033[0m: %s\n", err )
				break
			}
			
			str = str[0:len(str)-2]
			
			// Send it back on the read channel
			if debug {
				fmt.Printf( "<- \033[34;1m%s\033[0m\n", str )
			}
			read <- str
		}
	}()
	
	go func() {
		for {
			str := <-write
			
			// Note it
			if debug {
				fmt.Printf( "-> \033[35;1m%s\033[0m\n", str )
			}
			
			_, err := writer.WriteString( str + "\r\n" )
			if err != nil {
				fmt.Fprintf( os.Stderr, "\033[31;1mERROR\033[0m: %s\n", err )
				break
			}
			
			writer.Flush()
		}
	}()
	
	// Connect
	write <- "NICK " + nick
	write <- "USER " + nick + " * * :rchowe's bot"
	for c := range channels {
		write <- "JOIN " + channels[c]
	}
	
	// To make foonetic admins happy
	write <- "MODE +Bix carbon"
	
	for {
		str := <- read
		if str[0:6] == "PING :" {
			if debug {
				fmt.Printf( "\033[34mSERVER PING\033[0m\n" )
			}
			write <- "PONG :" + str[7:len(str)-1]
		}
		
		data := privmsgRegexp.FindAllStringSubmatch( str, 1 )
		if data != nil {
						
			user_ := User{ data[0][1], data[0][2], data[0][3] }
			message := &Message{ user_, data[0][4], data[0][5], false }
			if len(message.Content) > 9 && message.Content[0:8] == "\001ACTION " {
				message.Content = message.Content[8:len(message.Content)-1]
				message.Action = true
			}
			
			if message.Action {
				fmt.Printf( "[\033[34m%s\033[0m] ** \033[4;1m%s\033[0m \033[1m%s\033[0m **\n", message.Channel, message.User.Nick, message.Content )
			} else if message.Content == "\001VERSION\001" {
				write <- "carbon v1.0"
			} else {
				fmt.Printf( "[\033[34m%s\033[0m] \033[4;1m%s\033[0m: %s\n", message.Channel, message.User.Nick, message.Content )
			}
		}
		
		data = modeRegexp.FindAllStringSubmatch( str, 1 )
		if data != nil {
			user_   := User{ data[0][1], data[0][2], data[0][3] }
			message := &Message{ user_, data[0][4], data[0][5], false }
			
			fmt.Printf( "[\033[34m%s\033[0m] \033[4;1m%s\033[0m changed mode to \033[32m%s\033[0m\n", message.Channel, message.User.Nick, message.Content )
		}
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/bmc-toolbox/bmclib/v2"
	"github.com/go-logr/logr"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	user := flag.String("user", "", "BMC username, required")
	pass := flag.String("password", "", "BMC password, required")
	host := flag.String("host", "", "BMC hostname or IP address, required")
	isoURL := flag.String("iso", "", "The HTTP URL to the ISO to be mounted, leave empty to unmount")
	flag.Parse()

	if *user == "" {
		u := os.Getenv("BMC_USER")
		if u == "" {
			fmt.Fprintln(os.Stderr, "user arg or BMC_USER env is required")
			flag.PrintDefaults()
			os.Exit(1)
		}
		*user = u
	}
	if *pass == "" {
		p := os.Getenv("BMC_PASS")
		if p == "" {
			fmt.Fprintln(os.Stderr, "pass arg or BMC_PASSWORD env is required")
			flag.PrintDefaults()
			os.Exit(1)
		}
		*pass = p
	}
	if *host == "" {
		h := os.Getenv("BMC_HOST")
		if h == "" {
			fmt.Fprintln(os.Stderr, "host arg or BMC_HOST env is required")
			flag.PrintDefaults()
			os.Exit(1)
		}
		*host = h
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true}))
	log := logr.FromSlogHandler(l.Handler())

	cl := bmclib.NewClient(*host, *user, *pass, bmclib.WithLogger(log))
	if err := cl.Open(ctx); err != nil {
		panic(err)
	}
	defer cl.Close(ctx)

	if *isoURL == "" {
		medias, err := cl.GetVirtualMedia(ctx)
		if err != nil {
			log.Info("debugging", "metadata", cl.GetMetadata())
			panic(err)
		}
		log.Info("virtual media operation successful", "metadata", cl.GetMetadata())
		fmt.Println(medias)
		for _, media := range medias {
			fmt.Printf("\n{\n")
			fmt.Println(" Image:", media.Image)
			fmt.Println(" MediaType:", media.MediaType)
			fmt.Println(" UserName:", media.UserName)
			fmt.Println(" Password:", media.Password)
			fmt.Println(" Inserted:", media.Inserted)
			fmt.Println(" WriteProtected:", media.WriteProtected)
			fmt.Printf("}\n")
		}
	} else {
		ok, err := cl.SetVirtualMedia(ctx, "CD", *isoURL)
		if err != nil {
			log.Info("debugging", "metadata", cl.GetMetadata())
			panic(err)
		}
		if !ok {
			log.Info("debugging", "metadata", cl.GetMetadata())
			panic("failed virtual media operation")
		}
		log.Info("virtual media operation successful", "metadata", cl.GetMetadata())
	}
}

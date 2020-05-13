package main

import (
	"context"
	"fmt"
	"github.com/digitalocean/godo"
	"os"
)

func main() {
	var newDroplet DropletCreateRequest
	var dropletIP string
	var dropletID int

	client := godo.NewFromToken("DO_API_KEY")

	if len(os.Args) < 2 {
		fmt.Println("Usage: ./do <droplet name>")
		return
	}
	dropletName := os.Args[1]

	ctx := context.TODO()
	droplet, _, err := client.Droplets.List(ctx, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	for i := range droplet {
		if droplet[i].Name == dropletName {
			dropletIP = droplet[i].Networks.V4[0].IPAddress
			dropletID = droplet[i].ID
			newDroplet.Name = dropletName
			newDroplet.Region = droplet[i].Region
			break
		}
	}

	if dropletIP == "" {
		fmt.Printf("Error: Couldn't find droplet %s\n", dropletName)
		return
	}

	fmt.Printf("Found droplets:\n-- %s (%d) IP: %s\n", dropletName, dropletID, dropletIP)

	snapshots, _, err := client.Snapshots.List(ctx, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Snapshots:\n")
	for i := range snapshots {
		fmt.Printf("-- %s (%s) - %s - %s\n", snapshots[i].Name, snapshots[i].ID, snapshots[i].ResourceType, snapshots[i].Regions)
	}

	fmt.Printf("Volumes:\n")
	volumes, _, err := client.Storage.ListVolumes(ctx, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	for i := range volumes {
		fmt.Printf("-- %s (%s)- Droplet ID: %d\n", volumes[i].Name, volumes[i].ID, volumes[i].DropletIDs[0])
	}

	fmt.Printf("Creating new snapshot for droplet: %s..\n", dropletName)
	action, _, err := client.DropletActions.Snapshot(ctx, dropletID, dropletName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if dropletWait(ctx, client, dropletID, action.ID) == true {
		fmt.Printf("-- Completed\n")
	}

	fmt.Printf("Disconnecting volumes from droplet %s..\n", dropletName)
	for i := range volumes {
		if volumes[i].DropletIDs[0] == dropletID {
			fmt.Printf("-- %s - ", volumes[i].Name)
			action, _, err := client.StorageActions.DetachByDropletID(ctx, volumes[i].ID, dropletID)
			if err != nil {
				fmt.Println("Error: %v\n", err)
				return
			}
			if dropletWait(ctx, client, dropletID, action.ID) == true {
				fmt.Printf("-- Detached\n")
				break
			}
		}
	}

	fmt.Printf("Deleting and recreating droplet from snapshot..\n")
	resp, err := client.DropletActions.Delete(ctx, dropletID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	/*
		droplet, resp, err := client.DropletActions.Create(
	*/

	fmt.Printf("Done\n")
	return
}

func dropletWait(ctx context.Context, client *godo.Client, dropletID int, actionID int) bool {
	for {
		action, _, err := client.DropletActions.Get(ctx, dropletID, actionID)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			break
		}

		if action.Status == "completed" || action.Status == "errored" {
			return true
		}
	}
	return false
}

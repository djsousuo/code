package main

import (
        "context"
        "fmt"
        "os"
        "github.com/Azure/azure-sdk-for-go/services/trafficmanager/mgmt/2018-04-01/trafficmanager"
        "github.com/Azure/go-autorest/autorest/azure/auth"
        "github.com/Azure/go-autorest/autorest/to"
)

func main() {
        subscription := "YOUR-API-SUBSCRIPTION-ID"

        tmClient := trafficmanager.NewProfilesClient(subscription)
        ctx := context.Background()
        authorizer, err := auth.NewAuthorizerFromEnvironment()
        if err == nil {
                tmClient.Authorizer = authorizer
        }


        available, err := tmClient.CheckTrafficManagerRelativeDNSNameAvailability(ctx, trafficmanager.CheckTrafficManagerRelativeDNSNameAvailabilityParameters{
                Name: to.StringPtr(os.Args[1]),
                Type: to.StringPtr("Microsoft.Network/trafficManagerProfiles"),
        })
        if err != nil {
                fmt.Printf("Can't check availability: %v", err)
                return
        }

        fmt.Printf("%s.trafficmanager.net - %v\n", os.Args[1], *available.NameAvailable)
}

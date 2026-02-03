package main

import (
	"context"
	"fmt"
	"log"
	"os"

	billing "cloud.google.com/go/billing/apiv1"
	"google.golang.org/api/option"
	billingpb "google.golang.org/genproto/googleapis/cloud/billing/v1"
)

func main() {
	fmt.Println("=== SlideForge Billing API Tester ===")

	ctx := context.Background()
	keyPath := "opt/mksauth-b258a892cd16.json"

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		log.Fatalf("Service account key not found at %s", keyPath)
	}

	// 1. Create Billing Client
	client, err := billing.NewCloudBillingClient(ctx, option.WithCredentialsFile(keyPath))
	if err != nil {
		log.Fatalf("Failed to create billing client: %v", err)
	}
	defer client.Close()

	// 2. List Billing Accounts
	fmt.Println("Listing billing accounts...")
	it := client.ListBillingAccounts(ctx, &billingpb.ListBillingAccountsRequest{})
	for {
		account, err := it.Next()
		if err != nil {
			fmt.Printf("End of list or error: %v\n", err)
			break
		}
		fmt.Printf("- Name: %s, Open: %v, Display Name: %s\n", account.Name, account.Open, account.DisplayName)

		// 3. For each account, try to get project billing info
		// Note: You need the project ID. Let's assume mksauth based on the service account name.
		projectID := "mksauth" // Based on user's mention
		fmt.Printf("Checking billing info for project: %s\n", projectID)
		info, err := client.GetProjectBillingInfo(ctx, &billingpb.GetProjectBillingInfoRequest{
			Name: "projects/" + projectID,
		})
		if err != nil {
			fmt.Printf("  ❌ Could not get project billing info: %v\n", err)
		} else {
			fmt.Printf("  ✅ Project %s is linked to billing account %s\n", projectID, info.BillingAccountName)
		}
	}
}

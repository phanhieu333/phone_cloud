package main

import (
	"fmt"

	"autogetjs/internal/device"
	"autogetjs/internal/job"
	"autogetjs/pkg/logger"
)

func main() {
	logger.Info("Starting AutoGetJS...")

	devices, err := device.GetDevices([]string{""})
	if err != nil {
		logger.Error("Failed to get devices: %v", err)
		return
	}

	links := []string{
		"https://savvysc.com/p/apparecchi-invisibili-convenienti-guida-completa-per-un-sorriso-allineato-a-costi-contenuti-524280.webm?__bt=true&_bot=2&_tag=581456_129_40_2353567_0&_type=index_link&ad_id=%7B%7Bad.id%7D%7D&ad_name=%7B%7Bad.name%7D%7D&arb_ad_id=2353567&arb_ad_id=2353567&arb_campaign_id=581456&arb_creative_id=2353567&arb_direct=on&campaign_id=%7B%7Bcampaign.id%7D%7D&campaign_name=%7B%7Bcampaign.name%7D%7D&dontRedirect=true&network=facebook&section_id=%7B%7Badset.id%7D%7D&section_name=%7B%7Badset.name%7D%7D&short_name=fbk&utm_campaign=arb-581456&utm_source=fb",
		"https://tipssearch.com/p/understanding-retirement-monthly-income-a-guide-to-averages-and-age-based-variations-545886.webm?__bt=true&_bot=2&_tag=593509_130_40_2384674_0&_type=index_link&ad_id=%7B%7Bad.id%7D%7D&ad_name=%7B%7Bad.name%7D%7D&arb_ad_id=2384674&arb_ad_id=2384674&arb_campaign_id=593509&arb_creative_id=2384674&arb_direct=on&campaign_id=%7B%7Bcampaign.id%7D%7D&campaign_name=%7B%7Bcampaign.name%7D%7D&dontRedirect=true&network=facebook&section_id=%7B%7Badset.id%7D%7D&section_name=%7B%7Badset.name%7D%7D&short_name=fbk&utm_campaign=arb-593509&utm_source=fb",
		"https://searchcritic.com/p/a-complete-guide-to-dumpster-trailer-rental-sizes-benefits-and-how-to-choose-501548.webm?__bt=true&_bot=2&_tag=563798_131_40_2314392_0&_type=index_link&ad_id=%7B%7Bad.id%7D%7D&ad_name=%7B%7Bad.name%7D%7D&arb_ad_id=2314392&arb_ad_id=2314392&arb_campaign_id=563798&arb_creative_id=2314392&arb_direct=on&campaign_id=%7B%7Bcampaign.id%7D%7D&campaign_name=%7B%7Bcampaign.name%7D%7D&dontRedirect=true&network=facebook&section_id=%7B%7Badset.id%7D%7D&section_name=%7B%7Badset.name%7D%7D&short_name=fbk&utm_campaign=arb-563798&utm_source=fb",
		"https://savvysc.com/p/a-comprehensive-guide-to-magnesium-intake-guidelines-and-personalized-dosage-533862.webm?__bt=true&_bot=2&_tag=586244_129_40_2365898_0&_type=index_link&ad_id=%7B%7Bad.id%7D%7D&ad_name=%7B%7Bad.name%7D%7D&arb_ad_id=2365898&arb_ad_id=2365898&arb_campaign_id=586244&arb_creative_id=2365898&arb_direct=on&campaign_id=%7B%7Bcampaign.id%7D%7D&campaign_name=%7B%7Bcampaign.name%7D%7D&dontRedirect=true&network=facebook&section_id=%7B%7Badset.id%7D%7D&section_name=%7B%7Badset.name%7D%7D&short_name=fbk&utm_campaign=arb-586244&utm_source=fb",
		"https://smartdealsearch.com/p/unsold-toyota-vehicles-overview-pricing-availability-and-key-factors-surrounding-surplus-inventory-517669.webm?__bt=true&_bot=2&_tag=577429_185_2387_2344198_0&_type=index_link&ad_id=%7B%7Bad.id%7D%7D&ad_name=%7B%7Bad.name%7D%7D&arb_ad_id=2344198&arb_ad_id=2344198&arb_campaign_id=577429&arb_creative_id=2344198&arb_direct=on&campaign_id=%7B%7Bcampaign.id%7D%7D&campaign_name=%7B%7Bcampaign.name%7D%7D&dontRedirect=true&network=facebook&section_id=%7B%7Badset.id%7D%7D&section_name=%7B%7Badset.name%7D%7D&short_name=fbk&utm_campaign=arb-577429&utm_source=fb",
	}

	job.Run(devices, links)
	fmt.Println("Done.")
}

bin:
	@go build .
aws-ranges:
	@wget -O aws-ranges.json https://ip-ranges.amazonaws.com/ip-ranges.json
goog-ranges:
	@wget -O goog-ranges.json https://www.gstatic.com/ipranges/cloud.json
azure-ranges:
	@echo "Azure ranges must be manually downloaded because Microsoft hates automation I guess."
	@echo "Goto: https://www.microsoft.com/en-us/download/details.aspx?id=56519 and save it as azure-ranges.json"
#	@wget -O azure-ranges.json https://download.microsoft.com/download/7/1/D/71D86715-5596-4529-9B13-DA13A5DE5B63/ServiceTags_Public_20220117.json 
oracle-ranges:
	@wget -O oracle-ranges.json https://docs.oracle.com/en-us/iaas/tools/public_ip_ranges.json 

ranges: aws-ranges goog-ranges oracle-ranges 

clean:
	rm ./rangechk

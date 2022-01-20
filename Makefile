bin:
	@go build .
aws-ranges: aws-ranges.json
	@wget -O aws-ranges.json https://ip-ranges.amazonaws.com/ip-ranges.json
goog-ranges: goog-ranges.json
	@wget -O goog-ranges.json https://www.gstatic.com/ipranges/cloud.json
azure-ranges: azure-ranges.json
	@wget -O azure-ranges.json https://download.microsoft.com/download/7/1/D/71D86715-5596-4529-9B13-DA13A5DE5B63/ServiceTags_Public_20220117.json 
ranges: aws-ranges goog-ranges azure-ranges

clean:
	rm ./rangechk

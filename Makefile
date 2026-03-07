.PHONY: fuzz fuzz-helium fuzz-xpath fuzz-c14n fuzz-xsd fuzz-relaxng fuzz-html

FUZZTIME ?= 30s

fuzz: fuzz-helium fuzz-xpath fuzz-c14n fuzz-xsd fuzz-relaxng fuzz-html

fuzz-helium:
	go test ./ -run "^$$" -fuzz ^FuzzParse$$ -fuzztime $(FUZZTIME)
	go test ./ -run "^$$" -fuzz ^FuzzParseRoundtrip$$ -fuzztime $(FUZZTIME)

fuzz-xpath:
	go test ./xpath/ -run "^$$" -fuzz ^FuzzCompile$$ -fuzztime $(FUZZTIME)

fuzz-c14n:
	go test ./c14n/ -run "^$$" -fuzz ^FuzzCanonicalize$$ -fuzztime $(FUZZTIME)

fuzz-xsd:
	go test ./xsd/ -run "^$$" -fuzz ^FuzzCompile$$ -fuzztime $(FUZZTIME)
	go test ./xsd/ -run "^$$" -fuzz ^FuzzValidate$$ -fuzztime $(FUZZTIME)

fuzz-relaxng:
	go test ./relaxng/ -run "^$$" -fuzz ^FuzzCompile$$ -fuzztime $(FUZZTIME)
	go test ./relaxng/ -run "^$$" -fuzz ^FuzzValidate$$ -fuzztime $(FUZZTIME)

fuzz-html:
	go test ./html/ -run "^$$" -fuzz ^FuzzParse$$ -fuzztime $(FUZZTIME)

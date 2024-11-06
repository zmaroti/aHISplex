VERSION = 1.0
INST_DIR=~/bin

.PHONY: build
build:
	go mod download
	go build -o bin/transToHISplex     ./cmd/transToHISplex/
	go build -o bin/classifHISplex     ./cmd/classifHISplex/

install:
	install -D -t $(INST_DIR) bin/transToHISplex bin/classifHISplex bin/aHISplex.sh
	install -d  $(INST_DIR)/aHISplex_data/  $(INST_DIR)/aHISplex_data/ref_GRCh37  $(INST_DIR)/aHISplex_data/ref_GRCh38  $(INST_DIR)/aHISplex_data/ref_hg19
	install -m 644 -t $(INST_DIR)/aHISplex_data/ aHISplex_data/*.txt
	install -m 644 -t $(INST_DIR)/aHISplex_data/ aHISplex_data/*.tsv
	install -m 644 -t $(INST_DIR)/aHISplex_data/ref_GRCh37 aHISplex_data/ref_GRCh37/*.bin
	install -m 644 -t $(INST_DIR)/aHISplex_data/ref_GRCh38 aHISplex_data/ref_GRCh38/*.bin
	install -m 644 -t $(INST_DIR)/aHISplex_data/ref_hg19 aHISplex_data/ref_hg19/*.bin


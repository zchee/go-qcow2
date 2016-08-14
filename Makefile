IGNORE_FILE = $(foreach file,Makefile,--ignore $(file))
IGNORE_DIR = $(foreach dir,vendor testdata internal,--ignore-dir $(dir))
IGNORE = $(IGNORE_FILE) $(IGNORE_DIR)

todo: 
	@ag 'TODO(\(.+\):|:)' --after=1 $(IGNORE) || true
	@ag 'BUG(\(.+\):|:)' --after=1 $(IGNORE)|| true
	@ag 'XXX(\(.+\):|:)' --after=1 $(IGNORE)|| true
	@ag 'FIXME(\(.+\):|:)' --after=1 $(IGNORE) || true
	@ag 'NOTE(\(.+\):|:)' --after=1 $(IGNORE) || true

.PHONY: todo

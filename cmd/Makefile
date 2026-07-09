DIRS += $(shell find */* -maxdepth 0 -name Makefile -exec dirname "{}" \;)

default: precommit

test:
	@for i in $(DIRS); do \
		cd $$i; \
		echo "test $${i}"; \
		make test; \
		cd ..; \
	done

precommit:
	@for i in $(DIRS); do \
		cd $$i; \
		echo "precommit $${i}"; \
		make precommit; \
		cd ..; \
	done

apply:
	@for i in $(DIRS); do \
		cd $$i; \
		echo "apply $${i}"; \
		make apply; \
		cd ..; \
	done

ensure:
	@for i in $(DIRS); do \
		cd $$i; \
		echo "ensure $${i}"; \
		make ensure; \
		cd ..; \
	done

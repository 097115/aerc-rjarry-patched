# This file is only left here for explicit error about GNU make requirement
# when building with other make flavours.
#
# Do not edit this file. Edit GNUmakefile instead.
.PHONY: all
all .DEFAULT:
	@echo "Please build and install using GNU make (gmake)"; exit 1

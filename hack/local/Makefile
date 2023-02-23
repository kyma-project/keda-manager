.PHONY: run-k3d-with-keda
run-k3d-with-keda:
	@make -C ./keda run

.PHONY: busola-enable
busola-enable:
	@make -C ./busola busola-enable

.PHONY: verify
verify:
	@make -C ./keda verify

.PHONY: run
run: run-k3d-with-keda busola-enable

.PHONY: test
test: run verify
	@echo '### keda operator opertational'

.PHONY: stop
stop:
	@make -C ./keda stop

.PHONY: upgrade-to
upgrade-to:
	@make -C ./keda reinstall-keda

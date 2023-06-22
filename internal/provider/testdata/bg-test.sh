#!/usr/bin/env bash

container_name="hubble-ui-frontened-nginx-init-check-$(date +%s)"
docker run --name="${container_name}" "${IMAGE_NAME}" &
trap 'docker rm -f "${container_name}"' ERR

# Give the container fifteen seconds to create and configure
# timeout function may not be available on infra, so until check + install is added we'll do manually
lookup_config_text="using the \"epoll\" event method"
config_text_present=false
for i in {1..15}; do
  if ! docker logs "${container_name}" | grep -i "${lookup_config_text}"; then sleep 1;
  else config_text_present=true; break;
  fi
done
if ! "${config_text_present}"; then exit 1; fi

# Force container removal
docker rm -f "${container_name}"

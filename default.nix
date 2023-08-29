{ pkgs ? import <nixpkgs> {} }:
with pkgs;
buildGoModule rec {
  pname = "prometheus-steam-web-api-exporter";
  version = "dev";

  src = {
    outPath = ./.;
  };

  vendorHash = "sha256-yGv3zPAO844DHpV7iGe4KGhLAbyfMqaC3gGJCVZm4U4=";
}

{ pkgs ? import <nixpkgs> {} }:
with pkgs;
buildGoModule rec {
  pname = "prometheus-steam-web-api-exporter";
  version = "dev";

  src = {
    outPath = ./.;
  };

  vendorHash = "sha256-wYq4cuKc7w8UoWG9OCuX2SotIcYJ/JHNQXnzl3cTyxM=";
}

{
  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs =
    {
      self,
      flake-utils,
      nixpkgs,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };

        outrig-server = pkgs.callPackage ./server/outrig-server.nix { };
      in
      {
        defaultPackage = outrig-server;

        overlays.default = final: prev: {
          outrig = self.outputs.defaultPackage.${prev.system};
        };
      }
    );
}

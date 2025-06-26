{
  description = "MPLI documentation flake";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
    in
    {
      devShells = builtins.listToAttrs (
        map (
          system:
          let
            pkgs = import nixpkgs { inherit system; };
          in
          {
            name = system;
            value.default = pkgs.mkShell {
              packages = [
                pkgs.go
                pkgs.net-snmp
                pkgs.gnumake
                pkgs.plantuml
                pkgs.goose
                pkgs.sqlc
                pkgs.operator-sdk
              ];
            };
          }
        ) systems
      );
    };
}

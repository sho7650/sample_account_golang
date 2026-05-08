{
  description = "sample_account — synthetic Japanese personal-account record generator (Go port)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        goVersion = pkgs.go_1_26;
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "sample_account";
          version = "0.1.0";
          src = ./.;
          vendorHash = null;
          subPackages = [ "cmd/sample_account" ];
          env = { CGO_ENABLED = "0"; };
          ldflags = [ "-s" "-w" ];
          meta = {
            description = "Synthetic Japanese personal-account record generator (Go port of sample_account)";
            mainProgram = "sample_account";
            license = pkgs.lib.licenses.mit;
          };
        };

        devShells.default = pkgs.mkShell {
          packages = [
            goVersion
            pkgs.gopls
            pkgs.gofumpt
            pkgs.golangci-lint
            pkgs.delve
            pkgs.gnumake
          ];
          shellHook = ''
            export GOROOT="${goVersion}/share/go"
            export TZ="Asia/Tokyo"
            echo "sample_account dev shell — go $(${goVersion}/bin/go version | awk '{print $3}')"
          '';
        };

        formatter = pkgs.nixpkgs-fmt;
      });
}

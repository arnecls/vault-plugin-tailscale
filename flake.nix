# Borrowed from vault-plugin-secrets-github.
# Needs to be cleaned up and adapted to this plugin
# The original project used make to build and release the plugin.
{
  description = "vault-plugin-secrets-github";

  inputs = {
    nixpkgs.url = github:NixOS/nixpkgs/nixos-unstable;
    flake-parts.url = github:hercules-ci/flake-parts;

    devshell = {
      url = github:numtide/devshell;
      inputs.nixpkgs.follows = "nixpkgs";
    };

    gomod2nix = {
      url = github:nix-community/gomod2nix;
      inputs.nixpkgs.follows = "nixpkgs";
    };

    gitignore = {
      url = github:hercules-ci/gitignore.nix;
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{ self
    , nixpkgs
    , flake-parts
    , devshell
    , gomod2nix
    , gitignore
    , ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } ({
      systems = [
        "x86_64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
        "aarch64-linux"
      ];

      perSystem = { config, pkgs, system, ... }:
        let
          name = "vault-plugin-tailscale";
          package = "github.com/arnecls/${name}";
          rev = self.rev or "dirty";
          ver = if self ? "dirtyRev" then self.dirtyShortRev else self.shortRev;
          date = self.lastModifiedDate or "19700101";
          go = pkgs.go_1_24;
        in
        rec
        {
          _module.args.pkgs = import nixpkgs {
            inherit system;
            config.allowUnfree = true; # BSL2... Hashicorp...
            overlays = [ devshell.overlays.default gomod2nix.overlays.default ];
          };

          packages.default =
            gomod2nix.legacyPackages.${system}.buildGoApplication {
              inherit name go;
              src = gitignore.lib.gitignoreSource ./.;
              # Must be added due to bug:
              # https://github.com/nix-community/gomod2nix/issues/120
              pwd = ./.;
              flags = [ "-trimpath" ];
              ldflags = [
                "-s"
                "-w"
                "-extld ld"
                "-extldflags"
                "-static"
                "-X ${package}/github.projectName=${name}"
                "-X ${package}/github.projectDocs=https://${package}"
              ];
              doCheck = false;
            };

          devShells.default = pkgs.devshell.mkShell rec {
            inherit name;

            motd = builtins.concatStringsSep "\n" [
              "{2}${ name}{reset}"
              "menu                              - available commands"
            ];

            env = [
              { name = "VAULT_ADDR"; value = "http://127.0.0.1:8200"; }
            ];

            packages = with pkgs; [
              bashInteractive
              coreutils
              gnugrep
              go
              golangci-lint
              gomod2nix.legacyPackages.${system}.gomod2nix
              goreleaser
              syft
              vault-bin
            ];

            commands = with pkgs; let prjRoot = "cd $PRJ_ROOT;"; in
            [
              {
                inherit name;
                command = "nix run";
                help = "build and run the project binary";
              }
              {
                name = "build";
                command = "nix build";
                help = "build and run the project binary";
              }
              {
                name = "tidy";
                command = prjRoot + ''
                  echo >&2 "==> Tidying modules"
                  go mod tidy && gomod2nix
                '';
                help = "clean transient files";
              }
              {
                name = "lint";
                command = prjRoot + ''
                  echo >&2 "==> Linting"
                  if [ -v CI ]; then
                    mkdir -p test
                    ${golangci-lint}/bin/golangci-lint run \
                        --out-format=checkstyle | tee test/checkstyle.xml
                  else
                    ${golangci-lint}/bin/golangci-lint run --fast
                  fi
                '';
                help = "lint the project (heavyweight when CI=true)";
              }
            ];
          };
        };
    });
}

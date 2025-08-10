{
  description = "Visualize open PRs and Issues in nixos/nixpkgs over time";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default";
    pre-commit-hooks.url = "github:cachix/git-hooks.nix";
  };

  outputs =
    {
      self,
      nixpkgs,
      systems,
      ...
    }@inputs:
    let
      forAllSystems =
        function: nixpkgs.lib.genAttrs (import systems) (system: function nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: {
        default = self.packages.${pkgs.system}.nixpkgs-github-timeline;
        nixpkgs-github-timeline = pkgs.buildGoModule {
          pname = "nixpkgs-github-timeline";
          version = "0.0.1";
          src = ./.;
          vendorHash = "sha256-v5sNz9CnR8raOgrGuRWVifphCRTSEnuM7s+MquoChPg=";
        };
      });
      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          inherit (self.checks.${pkgs.system}.pre-commit-check) shellHook;
          buildInputs = self.checks.${pkgs.system}.pre-commit-check.enabledPackages;
          packages = with pkgs; [
            go
          ];
        };
      });
      checks = forAllSystems (pkgs: {
        pre-commit-check = inputs.pre-commit-hooks.lib.${pkgs.system}.run {
          src = ./.;
          hooks = {
            nixfmt-rfc-style.enable = true;
          };
        };
      });
    };
}

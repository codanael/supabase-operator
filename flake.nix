{
  description = "Supabase Kubernetes Operator — manage Supabase instances declaratively on K8s";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    claude-code.url = "github:sadjow/claude-code-nix";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
    claude-code,
  }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ claude-code.overlays.default ];
          config.allowUnfreePredicate = pkg: builtins.elem (nixpkgs.lib.getName pkg) [
            "claude-code"
          ];
        };
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = [
            # Go toolchain
            pkgs.go
            pkgs.gopls
            pkgs.gotools
            pkgs.golangci-lint

            # Kubernetes operator development
            pkgs.operator-sdk
            pkgs.kubebuilder
            pkgs.kubernetes-controller-tools
            pkgs.kustomize
            pkgs.setup-envtest

            # Kubernetes tools
            pkgs.kubectl
            pkgs.kind
            pkgs.kubernetes-helm

            # Container build
            pkgs.docker
            pkgs.docker-compose

            # System tools
            pkgs.openssl
            pkgs.git
            pkgs.jq
            pkgs.curl
            pkgs.gnumake

            # Claude Code CLI
            pkgs.claude-code
          ];

          shellHook = ''
            echo "Supabase Operator dev environment loaded"
            echo "Go:           $(go version)"
            echo "Operator SDK: $(operator-sdk version --short 2>/dev/null || echo 'available')"
            echo "kubectl:      $(kubectl version --client --short 2>/dev/null || kubectl version --client 2>/dev/null | head -1)"
            echo "kind:         $(kind version)"
            export GOPATH="$HOME/go"
            export PATH="$GOPATH/bin:$PATH"
            export KUBEBUILDER_ASSETS="$(setup-envtest use -p path 2>/dev/null || echo '''')"
          '';
        };
      }
    );
}

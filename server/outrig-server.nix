{
  stdenv,
  lib,
  fetchurl,
}:
let
  system = lib.splitString "-" stdenv.hostPlatform.system;
  arch = lib.head system;
  platform =
    let
      platform = lib.head (lib.tail system);
    in
    if platform == "linux" then
      "Linux"
    else if platform == "darwin" then
      "Darwin"
    else
      platform;

  # Platform-specific hashes for v0.9.1
  hashes = {
    "x86_64-linux" = "88d530991d576187429a0f18baa3ff3e63f894d307e78e665ecad33dac4a64b3";
    "aarch64-linux" = "f3fc45b0bb664f12147a0f860ac2b8a3a0e5ea2ac403430273bd10ef5551df50";
    "x86_64-darwin" = "36a8f9d98c60d41893f9cb2975e9d3f435dd81ad8a728142d6a6dc5a6e119e4b";
    "aarch64-darwin" = "f3a7269ba2c3b8f27a3b1a2cf714b2aea2cecdb0fcdf89fbf6073b4a04ed7782";
  };
in
(stdenv.mkDerivation rec {
  pname = "outrig";
  version = "0.9.1";

  src = fetchurl {
    url = "https://github.com/outrigdev/outrig/releases/download/v${version}/${pname}_${version}_${platform}_${arch}.tar.gz";
    sha256 = hashes.${stdenv.hostPlatform.system};
  };

  sourceRoot = "${pname}_${version}_${platform}_${arch}";

  unpackPhase = ''
    tar -xzf $src
  '';

  dontBuild = true;

  installPhase = ''
    mkdir -p $out/bin
    cp ${pname} $out/bin/${pname}
    chmod +x $out/bin/${pname}
  '';

  meta = {
    description = "Dev-time observability tool for Go programs. Search logs, monitor goroutines, and track variables";
    homepage = "https://outrig.run/";
    license = lib.licenses.asl20;
    maintainers = with lib.maintainers; [ sawka ];
    mainProgram = "outrig";
  };
})

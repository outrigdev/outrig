{
  stdenv,
  lib,
  fetchzip,
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

  # Platform-specific hashes for v0.8.2
  hashes = {
    "x86_64-linux" = "sha256:a794bd7788123baf7b842ac9bfe2f8ea9d2dbdcbe5399792562efc3c66cba06c";
    "aarch64-linux" = "sha256:dd4960b8b272c99bd4a6b30f85e3dd99146c5ab3457ecbfda26dac75cad80ab5";
    "x86_64-darwin" = "sha256:356b01f184ebcc0862eb0b46d1241a36b23f2e67bc51bc30eb3a1186e145c23e";
    "aarch64-darwin" = "sha256:b723a6b79650ba201d19d5fa70871d43c6095bc6d383502d989a2f83c8d7bb0d";
  };
in
(stdenv.mkDerivation rec {
  pname = "outrig";
  version = "0.8.2";

  src = fetchzip {
    url = "https://github.com/outrigdev/outrig/releases/download/v${version}/${pname}_${version}_${platform}_${arch}.tar.gz";
    hash = hashes.${stdenv.hostPlatform.system};
  };

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

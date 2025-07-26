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

  # Platform-specific hashes for v0.9.0-beta.0
  hashes = {
    "x86_64-linux" = "9033da756102dc20294a35e5e908e77b8989248740ece9862d7b9c3eb1dcfec0";
    "aarch64-linux" = "57f94e368de7f5ea7f2dfcc831528c493abf68973f0d0900233ef11ac9169b12";
    "x86_64-darwin" = "054631a2edbf8b45d9407fed7433540af3f383b812ebdbdc6081814cd7590288";
    "aarch64-darwin" = "54df41dc73e2772542bd9e5513a80e47130fb42429044bf46f554072f22b77a6";
  };
in
(stdenv.mkDerivation rec {
  pname = "outrig";
  version = "0.9.0-beta.0";

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

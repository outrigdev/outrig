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

  # Platform-specific hashes for v0.9.0
  hashes = {
    "x86_64-linux" = "1ac379f2b87c051b8f70536a345015eb8d50b961e4040574d8e70b74533b1bae";
    "aarch64-linux" = "613dda1775fd712afd8ee5658c9169cf821ae4c04edd4ce60fc9dfb958a83122";
    "x86_64-darwin" = "bb0870c681bec991eda245ed9648ac5e41d18d7580e6227d3c818e7dc58662df";
    "aarch64-darwin" = "1b44ec64f5cf56a764abc8c6451ac26dc3f9ee426fe535e6d0034a703a5c18ae";
  };
in
(stdenv.mkDerivation rec {
  pname = "outrig";
  version = "0.9.0";

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

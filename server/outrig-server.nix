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
in
(stdenv.mkDerivation rec {
  pname = "outrig";
  version = "0.8.2";

  src = fetchzip {
    url = "https://github.com/outrigdev/outrig/releases/download/v${version}/${pname}_${version}_${platform}_${arch}.tar.gz";
    hash = "sha256-0oJmBJCRCvY2ByFA7MosQ0y+065KvnUJNTczVRfqcF4=";
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

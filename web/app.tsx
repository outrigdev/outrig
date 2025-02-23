import reactLogo from "./assets/react.svg";
import viteLogo from "/vite.svg";
import "./app.css";
import { atom, useAtom } from "jotai";

const countAtom = atom(0);
countAtom.debugLabel = "countAtom";

function App() {
  const [count, setCount] = useAtom(countAtom);

  return (
    <>
      <div>
        <a href="https://vite.dev" target="_blank">
          <img src={viteLogo} className="logo" alt="Vite logo" />
        </a>
        <a href="https://react.dev" target="_blank">
          <img src={reactLogo} className="logo react" alt="React logo" />
        </a>
      </div>
      <h1>Vite + React</h1>
      <div className="card">
        <button onClick={() => setCount((count) => count + 1)}>
          count is {count}
        </button>
        <p>
          Edit <code>web/App.tsx</code> and save to test HMR
        </p>
      </div>
      <p className="read-the-docs">
        Click on the{" "}
        <span className="text-bold text-blue-500">Vite and React</span> logos to
        learn more
      </p>
    </>
  );
}

export default App;

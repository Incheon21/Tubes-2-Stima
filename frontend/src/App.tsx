import './App.css';
import RecipeExplorer from '../components/RecipeExplorer';
import { Toaster } from 'react-hot-toast';

function App() {
  return (
    <div className="App">
      <Toaster position="top-center" />
      <h1 className="text-3xl font-bold mb-6">Little Alchemy 2 Recipe Explorer</h1>
      <RecipeExplorer />
    </div>
  );
}

export default App;
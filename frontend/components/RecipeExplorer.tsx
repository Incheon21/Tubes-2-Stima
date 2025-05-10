import { useState, useEffect } from 'react';
import axios from 'axios';
import toast from 'react-hot-toast';
import ControlPanel from './ControlPanel';
import VisualizationPanel from './VisualizationPanel';
import type { Algorithm, TreeType, ElementData, TreeData, PathNode } from '../types/types';

const RecipeExplorer = () => {
  const [serverUrl, setServerUrl] = useState<string>('http://localhost:8080');
  const [target, setTarget] = useState<string>('');
  const [treeType, setTreeType] = useState<TreeType>('best-recipes-tree');
  const [algorithm, setAlgorithm] = useState<Algorithm>('bfs');
  const [treeCount, setTreeCount] = useState<number>(3);
  
  const [logs, setLogs] = useState<{message: string, type: string}[]>([]);
  const [allElements, setAllElements] = useState<ElementData[]>([]);
  const [currentTrees, setCurrentTrees] = useState<TreeData[]>([]);
  const [currentTreeIndex, setCurrentTreeIndex] = useState<number>(0);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  
  const [stats, setStats] = useState({
    algorithm: '-',
    timeElapsed: 0,
    nodesVisited: 0,
    treesFound: 0
  });

  // Add a log message
  const addLog = (message: string, type: 'debug' | 'error' | 'success' = 'debug') => {
    setLogs(prev => [...prev, { message, type }]);
  };

  // Load elements
  const loadElements = async () => {
    try {
      addLog('Fetching all elements...');
      setIsLoading(true);
      
      // Backend endpoint: GET /api/elements
      // This should return an array of all available elements
      const response = await axios.get<ElementData[]>(`${serverUrl}/api/elements`);
      
      setAllElements(response.data);
      addLog(`Successfully loaded ${response.data.length} elements`, 'success');
      setIsLoading(false);
      return true;
    } catch (error) {
      setIsLoading(false);
      if (axios.isAxiosError(error)) {
        addLog(`Failed to fetch elements: ${error.message}`, 'error');
        toast.error(`Failed to load elements: ${error.message}`);
      } else {
        addLog(`Failed to fetch elements: Unknown error`, 'error');
        toast.error('Failed to load elements');
      }
      return false;
    }
  };

  // Test server connection
  const testConnection = async () => {
    try {
      addLog(`Testing connection to ${serverUrl}...`);
      setIsLoading(true);
      
      // Backend endpoint: GET /api/elements
      // Used here just to test if the backend is responding
      const response = await axios.get(`${serverUrl}/api/elements`);
      
      addLog(`Connection successful! Server is running at ${serverUrl}`, 'success');
      toast.success('Connection successful!');
      
      // Load elements if connection is successful
      await loadElements();
      setIsLoading(false);
    } catch (error) {
      setIsLoading(false);
      if (axios.isAxiosError(error)) {
        addLog(`Connection failed: ${error.message}`, 'error');
        toast.error(`Connection failed: ${error.message}`);
      } else {
        addLog(`Connection failed: Unknown error`, 'error');
        toast.error('Connection failed');
      }
      
      addLog('Please check that:', 'error');
      addLog('1. The server is running', 'error');
      addLog('2. The URL is correct', 'error');
      addLog('3. CORS is enabled on the server', 'error');
    }
  };

  // Visualize recipe trees
  const visualizeRecipes = async () => {
    if (!target) {
      addLog('Please enter a target element', 'error');
      toast.error('Please enter a target element');
      return;
    }
    
    try {
      addLog(`Fetching ${treeType} for ${target} using ${algorithm} algorithm...`);
      setIsLoading(true);
      
      let url: string;
      
      // Determine which endpoint to use based on the algorithm
      if (algorithm === 'bfs') {
        // Backend endpoint: GET /api/bfs/{target}?count={count}&singlePath=false
        // This should return BFS paths to create the target element
        url = `${serverUrl}/api/bfs/${encodeURIComponent(target)}?count=${treeCount}&singlePath=false`;
        addLog(`Using BFS endpoint: ${url}`);
      } else if (algorithm === 'dfs') {
        // Backend endpoint: GET /api/multiple-recipes/{target}?count={count}
        // This should return DFS paths to create the target element
        url = `${serverUrl}/api/dfs-tree/${encodeURIComponent(target)}?count=${treeCount}`;
        addLog(`Using DFS endpoint: ${url}`);
      } else if (algorithm === 'multithreaded-bfs') {
        // Backend endpoint: GET /api/mt-bfs/{target}?count={count}
        // This should return results from the multi-threaded BFS algorithm
        url = `${serverUrl}/api/mt-bfs/${encodeURIComponent(target)}?count=${treeCount}`;
        addLog(`Using multithreaded BFS endpoint: ${url}`);
      } else {
        // Backend endpoint: GET /api/{treeType}/{target}?count={count}&algorithm={algorithm}
        // General endpoint format for other algorithms
        url = `${serverUrl}/api/${treeType}/${encodeURIComponent(target)}?count=${treeCount}&algorithm=${algorithm}`;
        addLog(`Using default endpoint: ${url}`);
      }
      
      const response = await axios.get(url);
      const result = response.data;
      
      addLog('Data received successfully');
      setIsLoading(false);
      
      // Update stats
      setStats({
        algorithm: algorithm.toUpperCase(),
        timeElapsed: result.timeElapsed || 0,
        nodesVisited: result.nodesVisited || 0,
        treesFound: (result.trees?.length || result.paths?.length || 0)
      });
      
      // Handle the results
      handleResults(result);
    } catch (error) {
      setIsLoading(false);
      if (axios.isAxiosError(error)) {
        addLog(`Visualization failed: ${error.message}`, 'error');
        toast.error(`Failed to get recipes: ${error.message}`);
      } else {
        addLog(`Visualization failed: Unknown error`, 'error');
        toast.error('Failed to get recipes');
      }
    }
  };

  // Handle visualization results
  const handleResults = (result: any) => {
    // Paths format (BFS or DFS)
    if (result.paths && Array.isArray(result.paths)) {
      addLog(`Received ${result.paths.length} paths from ${algorithm}`);
      
      // Convert paths to trees for visualization
      const trees = result.paths.map((path: PathNode[]) => convertPathToTree(path, target));
      setCurrentTrees(trees);
      setStats(prev => ({ ...prev, treesFound: trees.length }));
      
      if (trees.length > 0) {
        setCurrentTreeIndex(0);
        addLog(`Visualizing ${algorithm} path ${1} of ${trees.length} for ${target}`, 'success');
      } else {
        addLog(`No paths found for ${target}`, 'error');
        toast.error(`No paths found for ${target}`);
      }
    } 
    // Trees format
    else if (result.trees && Array.isArray(result.trees)) {
      addLog(`Received ${result.trees.length} recipe trees`);
      setCurrentTrees(result.trees);
      setStats(prev => ({ ...prev, treesFound: result.trees.length }));
      
      if (result.trees.length > 0) {
        setCurrentTreeIndex(0);
        addLog(`Visualizing recipe tree 1 of ${result.trees.length} for ${target}`, 'success');
      } else {
        addLog('No recipe trees found', 'error');
        toast.error('No recipe trees found');
      }
    }
    // Single tree format
    else if (result.name || (result.Element || result.element)) {
      addLog(`Received single recipe tree for ${target}`);
      setCurrentTrees([result]);
      setStats(prev => ({ ...prev, treesFound: 1 }));
      setCurrentTreeIndex(0);
      addLog(`Visualizing single recipe tree for ${target}`, 'success');
    }
    // Empty or invalid results
    else {
      addLog(`No recipe data found for ${target}`, 'error');
      toast.error(`No recipe data found for ${target}`);
      setCurrentTrees([]);
      setStats(prev => ({ ...prev, treesFound: 0 }));
    }
  };

  // Convert path to tree
  const convertPathToTree = (path: PathNode[], targetElement: string): TreeData => {
    // Normalize property names for consistency
    const normalizedPath = path.map(node => ({
      Element: node.element || node.Element,
      ImagePath: node.imagePath || node.ImagePath,
      Ingredients: node.ingredients || node.Ingredients || []
    }));
    
    if (!normalizedPath || normalizedPath.length === 0) {
      return { name: targetElement, ingredients: [] };
    }
    
    // Helper to track visited elements to detect circular references
    const visitedInPath = new Set<string>();
    
    function buildTree(currentElement: string, remainingPath: any[]): TreeData {
      // Find the node for current element
      const currentNode = remainingPath.find(node => node.Element === currentElement);
      if (!currentNode) {
        return { name: currentElement, ingredients: [] };
      }
      
      // Check for circular reference
      if (visitedInPath.has(currentElement)) {
        return { 
          name: currentElement,
          imagePath: currentNode.ImagePath,
          isCircularReference: true,
          ingredients: [] 
        };
      }
      
      // Add to visited set for circular reference detection
      visitedInPath.add(currentElement);
      
      // Create the node for this element
      const node: TreeData = {
        name: currentElement,
        imagePath: currentNode.ImagePath,
        isBaseElement: ['Water', 'Fire', 'Earth', 'Air'].includes(currentElement),
        ingredients: []
      };
      
      // Process ingredients if any
      if (currentNode.Ingredients && currentNode.Ingredients.length > 0) {
        currentNode.Ingredients.forEach((ingredient: string) => {
          const ingredientTree = buildTree(ingredient, remainingPath);
          node.ingredients.push(ingredientTree);
        });
      }
      
      // Remove from visited set when backtracking
      visitedInPath.delete(currentElement);
      
      return node;
    }
    
    // Find the target element in the path
    const targetNode = normalizedPath.find(node => node.Element === targetElement) || 
                     normalizedPath[normalizedPath.length - 1];
    
    // Build tree starting from target element
    return buildTree(targetNode.Element || targetElement, normalizedPath);
  };

  // Clear visualization
  const clearVisualization = () => {
    setCurrentTrees([]);
    setCurrentTreeIndex(0);
    setStats({
      algorithm: '-',
      timeElapsed: 0,
      nodesVisited: 0,
      treesFound: 0
    });
    addLog('Visualization cleared');
  };

  // Load elements on mount
  useEffect(() => {
    loadElements().catch(err => {
      addLog('Could not automatically load elements on page load.');
      addLog('Use the "Test Connection" button to connect to the server.');
    });
  }, []);

  return (
  <div className="flex flex-col lg:flex-row gap-8">
    <ControlPanel 
      serverUrl={serverUrl}
      setServerUrl={setServerUrl}
      target={target}
      setTarget={setTarget}
      treeType={treeType}
      setTreeType={setTreeType}
      algorithm={algorithm}
      setAlgorithm={setAlgorithm}
      treeCount={treeCount}
      setTreeCount={setTreeCount}
      logs={logs}
      stats={stats}
      testConnection={testConnection}
      visualizeRecipes={visualizeRecipes}
      clearVisualization={clearVisualization}
      elements={allElements}
      isLoading={isLoading}
    />
    
    <VisualizationPanel 
      currentTrees={currentTrees}
      currentTreeIndex={currentTreeIndex}
      setCurrentTreeIndex={setCurrentTreeIndex}
      targetElement={target}
      algorithm={algorithm}
    />
  </div>
);
};

export default RecipeExplorer;
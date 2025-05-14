import { useState, useEffect } from 'react';
import axios from 'axios';
import toast from 'react-hot-toast';
import ControlPanel from './ControlPanel';
import VisualizationPanel from './VisualizationPanel';
import type { Algorithm, TreeType, ElementData, TreeData, PathNode } from '../types/types';

interface RecipeResult {
  paths?: PathNode[][];
  trees?: TreeData[];
  name?: string;
  Element?: string;
  element?: string;
  timeElapsed?: number;
  nodesVisited?: number;
  [key: string]: unknown;
}

const RecipeExplorer = () => {
  const [target, setTarget] = useState<string>('');
  const [treeType, setTreeType] = useState<TreeType>('best-recipes-tree');
  const [algorithm, setAlgorithm] = useState<Algorithm>('bfs');
  const [treeCount, setTreeCount] = useState<number>(3);
  
  const [allElements, setAllElements] = useState<ElementData[]>([]);
  const [currentTrees, setCurrentTrees] = useState<TreeData[]>([]);
  const [currentTreeIndex, setCurrentTreeIndex] = useState<number>(0);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  
  const [stats, setStats] = useState({
    algorithm: '-',
    timeElapsed: 0,
    nodesVisited: 0,
    treesFound: 0,
  });

  console.log("tes:");
  const serverUrl = import.meta.env.VITE_API_URL ?? 'http://localhost:8080/api';

  const loadElements = async () => {
    try {
      setIsLoading(true);
      const response = await axios.get<ElementData[]>(`${serverUrl}/elements/`);      
      console.log("API response:", response.data);
      setAllElements(response.data);
      setIsLoading(false);
      return true;
    } catch (error) {
      setIsLoading(false);
      if (axios.isAxiosError(error)) {
        toast.error(`Failed to load elements: ${error.message}`);
      } else {
        toast.error('Failed to load elements');
      }
      return false;
    }
  };

  const visualizeRecipes = async () => {
    if (!target) {
      toast.error('Please enter a target element');
      return;
    }
    
    try {
      setIsLoading(true);
      
      let url: string;
      
      if (algorithm === 'bfs') {
        url = `${serverUrl}/bfs-tree/${encodeURIComponent(target)}?count=${treeCount}&singlePath=false`;
      } else if (algorithm === 'dfs') {
        url = `${serverUrl}/dfs-tree/${encodeURIComponent(target)}?count=${treeCount}`;
      } else if (algorithm === 'bidire') {
        url = `${serverUrl}/bidirectional/${encodeURIComponent(target)}?count=${treeCount}&multithreaded=true&tree=true`;
      } else {
        url = `${serverUrl}/${treeType}/${encodeURIComponent(target)}?count=${treeCount}&algorithm=${algorithm}`;
      }
      
      const response = await axios.get<RecipeResult>(url);
      console.log("API response:", response.data);
      const result = response.data;
      
      setIsLoading(false);
      
      setStats({
        algorithm: algorithm.toUpperCase(),
        timeElapsed: result.timeElapsed || 0,
        nodesVisited: result.nodesVisited || 0,
        treesFound: (result.trees?.length || result.paths?.length || 0)
      });
      
      handleResults(result);
    } catch (error) {
      setIsLoading(false);
      if (axios.isAxiosError(error)) {
        toast.error(`Failed to get recipes: ${error.message}`);
      } else {
        toast.error('Failed to get recipes');
      }
    }
  };

  const handleResults = (result: RecipeResult) => {
    if (result.paths && Array.isArray(result.paths)) {
      const trees = result.paths.map((path: PathNode[]) => convertPathToTree(path, target));
      setCurrentTrees(trees);
      setStats(prev => ({ ...prev, treesFound: trees.length }));
      
      if (trees.length > 0) {
        setCurrentTreeIndex(0);
      } else {
        toast.error(`No paths found for ${target}`);
      }
    } 
    else if (result.trees && Array.isArray(result.trees)) {
      setCurrentTrees(result.trees);
      setStats(prev => ({ ...prev, treesFound: result.trees?.length || 0 }));
      
      if (result.trees.length > 0) {
        setCurrentTreeIndex(0);
      } else {
        toast.error('No recipe trees found');
      }
    }
    else if (result.name || (result.Element || result.element)) {
      setCurrentTrees([result as unknown as TreeData]);
      setStats(prev => ({ ...prev, treesFound: 1 }));
      setCurrentTreeIndex(0);
    }
    else {
      toast.error(`No recipe data found for ${target}`);
      setCurrentTrees([]);
      setStats(prev => ({ ...prev, treesFound: 0 }));
    }
  };

  const convertPathToTree = (path: PathNode[], targetElement: string): TreeData => {
    const normalizedPath = path.map(node => ({
      Element: node.element || node.Element || '',
      ImagePath: node.imagePath || node.ImagePath,
      Ingredients: node.ingredients || node.Ingredients || []
    }));
    
    if (!normalizedPath || normalizedPath.length === 0) {
      return { name: targetElement, ingredients: [] };
    }
    
    const visitedInPath = new Set<string>();
    
    function buildTree(currentElement: string, remainingPath: {Element: string, ImagePath?: string, Ingredients: string[]}[]): TreeData {
      const currentNode = remainingPath.find(node => node.Element === currentElement);
      if (!currentNode) {
        return { name: currentElement, ingredients: [] };
      }
      
      if (visitedInPath.has(currentElement)) {
        return { 
          name: currentElement,
          imagePath: currentNode.ImagePath,
          isCircularReference: true,
          ingredients: [] 
        };
      }
      
      visitedInPath.add(currentElement);
      
      const node: TreeData = {
        name: currentElement,
        imagePath: currentNode.ImagePath,
        isBaseElement: ['Water', 'Fire', 'Earth', 'Air'].includes(currentElement),
        ingredients: []
      };
      
      if (currentNode.Ingredients && currentNode.Ingredients.length > 0) {
        currentNode.Ingredients.forEach((ingredient: string) => {
          const ingredientTree = buildTree(ingredient, remainingPath);
          node.ingredients.push(ingredientTree);
        });
      }
      
      visitedInPath.delete(currentElement);
      
      return node;
    }
    
    const targetNode = normalizedPath.find(node => node.Element === targetElement) || 
                     normalizedPath[normalizedPath.length - 1];
    
    return buildTree(targetNode.Element || targetElement, normalizedPath);
  };

  const clearVisualization = () => {
    setCurrentTrees([]);
    setCurrentTreeIndex(0);
    setStats({
      algorithm: '-',
      timeElapsed: 0,
      nodesVisited: 0,
      treesFound: 0
    });
  };

  useEffect(() => {
    loadElements();
  }, []);

  return (
    <div className="flex flex-col lg:flex-row gap-8">
      <ControlPanel 
        target={target}
        setTarget={setTarget}
        treeType={treeType}
        setTreeType={setTreeType}
        algorithm={algorithm}
        setAlgorithm={setAlgorithm}
        treeCount={treeCount}
        setTreeCount={setTreeCount}
        stats={stats}
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
import React from 'react';
import type { TreeData, Algorithm } from '../types/types';

interface TreeDetailsProps {
  tree: TreeData;
  targetElement: string;
  algorithm: Algorithm;
}

const TreeDetails: React.FC<TreeDetailsProps> = ({ tree, targetElement, algorithm }) => {
  return (
    <div>
      <h3 className="text-lg font-semibold mb-4 text-gray-800 flex items-center">
        <svg className="w-5 h-5 mr-2 text-blue-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
          <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
        </svg>
        Recipe Details
      </h3>
      
      <div className="bg-gradient-to-br from-gray-50 to-blue-50 p-5 rounded-xl border border-blue-100 shadow-sm">
        <div className="flex items-center mb-6">
          <div className="bg-white p-3 rounded-full shadow-sm mr-4">
            {tree.imagePath ? (
              <img src={tree.imagePath} alt={tree.name} className="w-10 h-10" />
            ) : (
              <div className="w-10 h-10 bg-gray-200 rounded-full flex items-center justify-center text-gray-500 text-xs font-medium">
                {tree.name.substring(0, 2).toUpperCase()}
              </div>
            )}
          </div>
          <div>
            <span className="text-xl font-medium text-gray-800">{tree.name}</span>
            {tree.isBaseElement && (
              <span className="ml-2 px-2 py-1 bg-yellow-100 text-yellow-800 text-xs font-medium rounded-full">Base Element</span>
            )}
          </div>
        </div>
        
        {tree.ingredients.length > 0 ? (
          <div className="space-y-4">
            <div className="font-medium text-gray-700 flex items-center">
              <svg className="w-5 h-5 mr-2 text-blue-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
              </svg>
              Made from:
            </div>
            <div className="ml-4 bg-white p-4 rounded-lg border border-gray-200 shadow-sm">
              {renderRecipeTree(tree, 0)}
            </div>
          </div>
        ) : (
          <div className="bg-white p-4 rounded-lg border border-gray-200 text-gray-600 italic flex items-center">
            <svg className="w-5 h-5 mr-2 text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2h-1V9a1 1 0 00-1-1z" clipRule="evenodd" />
            </svg>
            {tree.isBaseElement ? 'This is a base element' : 'No ingredients (missing recipe)'}
          </div>
        )}
      </div>
      
      {/* Show additional path details for BFS and DFS algorithms */}
      {(algorithm === 'bfs' || algorithm === 'dfs') && tree.ingredients.length > 0 && (
        <div className="mt-6">
          <h4 className="text-gray-800 font-semibold mb-3 flex items-center">
            <svg className="w-5 h-5 mr-2 text-blue-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M12.293 5.293a1 1 0 011.414 0l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-2.293-2.293a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
            {algorithm.toUpperCase()} Path Summary
          </h4>
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-5">
            <div className="flex flex-wrap items-center gap-2">
              {renderShortPath(tree, algorithm)}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

// Helper function to render a recipe tree
const renderRecipeTree = (node: TreeData, depth: number) => {
  const baseElements = ['Water', 'Fire', 'Earth', 'Air'];
  
  return (
    <div className="mb-2" key={`${node.name}-${depth}`}>
      <div className="flex items-center">
        <div 
          className={`inline-flex items-center px-3 py-2 rounded-lg shadow-sm text-sm
            ${node.isBaseElement 
              ? 'bg-yellow-50 border border-yellow-200 text-yellow-800' 
              : depth === 0 
                ? 'bg-green-50 border border-green-200 text-green-800' 
                : 'bg-blue-50 border border-blue-200 text-blue-800'
          }`}
        >
          {node.imagePath && (
            <img src={node.imagePath} alt={node.name} className="w-5 h-5 mr-2" />
          )}
          {node.name}
        </div>
        
        {node.isCircularReference && (
          <span className="ml-3 text-orange-500 text-sm italic flex items-center">
            <svg className="w-4 h-4 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
            </svg>
            circular reference
          </span>
        )}
      </div>
      
      {node.ingredients?.length > 0 && (
        <div className="ml-8 mt-3 pl-4 border-l-2 border-blue-200">
          {node.ingredients.map(ingredient => renderRecipeTree(ingredient, depth + 1))}
        </div>
      )}
    </div>
  );
};

// Helper function to render a short path for BFS/DFS results
const renderShortPath = (tree: TreeData, algorithm: Algorithm) => {
  // The implementation of this function remains the same, but let's enhance the visual appearance
  const baseElements = ['Water', 'Fire', 'Earth', 'Air'];
  const findBaseElements = (node: TreeData): TreeData[] => {
    if (baseElements.includes(node.name) || node.ingredients.length === 0) {
      return [node];
    }
    
    let bases: TreeData[] = [];
    for (const ingredient of node.ingredients) {
      bases = [...bases, ...findBaseElements(ingredient)];
    }
    return bases;
  };
  
  const findKeyIntermediates = (node: TreeData): TreeData[] => {
    if (node.ingredients.length === 0) {
      return [];
    }
    
    let intermediates: TreeData[] = [];
    if (!baseElements.includes(node.name) && node.name !== tree.name) {
      intermediates.push(node);
    }
    
    for (const ingredient of node.ingredients) {
      intermediates = [...intermediates, ...findKeyIntermediates(ingredient)];
    }
    return intermediates;
  };
  
  const baseElementNodes = findBaseElements(tree);
  const intermediateNodes = findKeyIntermediates(tree);
  
  // Create path visualization
  return (
    <>
      {/* Base elements */}
      {baseElementNodes.map((node, i) => (
        <React.Fragment key={`base-${i}`}>
          <div className="bg-yellow-50 border border-yellow-200 px-3 py-2 rounded-lg flex items-center shadow-sm">
            {node.imagePath && (
              <img src={node.imagePath} alt={node.name} className="w-5 h-5 mr-2" />
            )}
            {node.name}
          </div>
          {i < baseElementNodes.length - 1 && (
            <div className="text-lg font-bold text-gray-500">+</div>
          )}
        </React.Fragment>
      ))}
      
      {/* Arrow */}
      {baseElementNodes.length > 0 && intermediateNodes.length > 0 && (
        <div className="flex items-center mx-3">
          <svg className="w-6 h-6 text-blue-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M12.293 5.293a1 1 0 011.414 0l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-2.293-2.293a1 1 0 010-1.414z" clipRule="evenodd" />
          </svg>
        </div>
      )}
      
      {/* Intermediates */}
      {intermediateNodes.slice(0, 3).map((node, i) => (
        <React.Fragment key={`inter-${i}`}>
          <div className="bg-blue-50 border border-blue-200 px-3 py-2 rounded-lg flex items-center shadow-sm">
            {node.imagePath && (
              <img src={node.imagePath} alt={node.name} className="w-5 h-5 mr-2" />
            )}
            {node.name}
          </div>
          {i < Math.min(2, intermediateNodes.length - 1) && (
            <div className="flex items-center mx-2">
              <svg className="w-5 h-5 text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M12.293 5.293a1 1 0 011.414 0l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-2.293-2.293a1 1 0 010-1.414z" clipRule="evenodd" />
              </svg>
            </div>
          )}
        </React.Fragment>
      ))}
      
      {/* Ellipsis for long paths */}
      {intermediateNodes.length > 3 && (
        <div className="flex items-center mx-3 px-3 py-1 bg-gray-100 rounded-lg text-gray-700 font-medium">
          ...
        </div>
      )}
      
      {/* Target element */}
      {(baseElementNodes.length > 0 || intermediateNodes.length > 0) && (
        <>
          <div className="flex items-center mx-3">
            <svg className="w-6 h-6 text-green-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M12.293 5.293a1 1 0 011.414 0l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-2.293-2.293a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
          </div>
          <div className="bg-green-50 border border-green-200 px-3 py-2 rounded-lg flex items-center shadow-sm">
            {tree.imagePath && (
              <img src={tree.imagePath} alt={tree.name} className="w-5 h-5 mr-2" />
            )}
            <span className="font-medium">{tree.name}</span>
          </div>
        </>
      )}
    </>
  );
};

export default TreeDetails;
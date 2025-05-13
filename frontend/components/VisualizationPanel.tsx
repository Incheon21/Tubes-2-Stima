import React, { useEffect, useRef } from 'react';
import * as d3 from 'd3';
import type { TreeData, Algorithm } from '../types/types';
import TreeSelector from './TreeSelector';
import TreeDetails from './TreeDetails';

interface VisualizationPanelProps {
  currentTrees: TreeData[];
  currentTreeIndex: number;
  setCurrentTreeIndex: (index: number) => void;
  targetElement: string;
  algorithm: Algorithm;
}

const VisualizationPanel: React.FC<VisualizationPanelProps> = ({
  currentTrees,
  currentTreeIndex,
  setCurrentTreeIndex,
  targetElement,
  algorithm
}) => {
  const visualizationRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (currentTrees.length > 0 && currentTreeIndex < currentTrees.length) {
      visualizeTree(currentTrees[currentTreeIndex]);
    }
  }, [currentTrees, currentTreeIndex]);

  const visualizeTree = (treeData: TreeData) => {
    if (!treeData || !visualizationRef.current) return;
    
    // Clear previous visualization
    d3.select(visualizationRef.current).selectAll("*").remove();
    
    // Set up dimensions
    const margin = {top: 40, right: 90, bottom: 50, left: 90};
    const width = visualizationRef.current.offsetWidth - margin.left - margin.right;
    const height = 500 - margin.top - margin.bottom;
    
    // Create SVG
    const svg = d3.select(visualizationRef.current)
      .append("svg")
      .attr("width", width + margin.left + margin.right)
      .attr("height", height + margin.top + margin.bottom)
      .append("g")
      .attr("transform", `translate(${margin.left},${margin.top})`);
    
    // Process the tree data
    const hierarchyData = {
      name: treeData.name,
      isBaseElement: treeData.isBaseElement,
      isCircularReference: treeData.isCircularReference,
      noRecipe: treeData.noRecipe,
      imagePath: treeData.imagePath,
      children: treeData.ingredients.map(ing => processNode(ing))
    };
    
    function processNode(node: TreeData) {
      return {
        name: node.name,
        isBaseElement: node.isBaseElement,
        isCircularReference: node.isCircularReference,
        noRecipe: node.noRecipe,
        imagePath: node.imagePath,
        children: node.ingredients ? node.ingredients.map(ing => processNode(ing)) : []
      };
    }
    
    // Create the tree layout
    const treeLayout = d3.tree().size([width, height]);
    
    // Create root node and calculate positions
    const root = d3.hierarchy(hierarchyData);
    treeLayout(root);
    
    // Draw links between nodes
    svg.selectAll(".link")
      .data(root.links())
      .enter()
      .append("path")
      .attr("class", "link")
      .attr("d", d3.linkVertical<any, any>()
        .x(d => d.x)
        .y(d => d.y))
      .style("fill", "none")
      .style("stroke", "#ccc")
      .style("stroke-width", "2px");
    
    // Create node groups
    const nodes = svg.selectAll(".node")
      .data(root.descendants())
      .enter()
      .append("g")
      .attr("class", "node")
      .attr("transform", d => `translate(${d.x},${d.y})`);
    
    // Add circles to nodes
    nodes.append("circle")
      .attr("r", 6)
      .style("fill", (d: any) => {
        if (d.data.isBaseElement) return "#FFEB3B"; // Yellow for base elements
        if (d.data.isCircularReference) return "#FF9800"; // Orange for circular references
        if (d.data.noRecipe) return "#E0E0E0"; // Gray for no recipe
        if (d.depth === 0) return "#4CAF50"; // Green for target element
        return "#2196F3"; // Blue for regular elements
      })
      .style("stroke", "#fff")
      .style("stroke-width", "1.5px");
    
    // Add text labels
    nodes.append("text")
      .attr("dy", ".35em")
      .attr("x", (d: any) => d.children ? -13 : 13)
      .attr("text-anchor", (d: any) => d.children ? "end" : "start")
      .text((d: any) => d.data.name)
      .style("font-size", "12px")
      .style("font-family", "sans-serif");
  };

  return (
    <div className="w-full lg:w-2/3 bg-white rounded-xl shadow-xl overflow-hidden border border-gray-100">
      <div className="bg-gradient-to-r from-blue-600 to-indigo-600 py-4 px-6">
        <h2 className="text-xl font-semibold text-white">Recipe Visualization</h2>
      </div>
      
      {currentTrees.length > 1 && (
        <div className="p-4 bg-blue-50 border-b border-blue-100">
          <TreeSelector 
            count={currentTrees.length} 
            currentIndex={currentTreeIndex} 
            setCurrentIndex={setCurrentTreeIndex} 
          />
        </div>
      )}
      
      <div 
        ref={visualizationRef} 
        className="w-full border-b border-gray-200 overflow-auto bg-gradient-to-br from-gray-50 to-white"
        style={{ height: '500px' }}
      >
        {currentTrees.length === 0 && (
          <div className="flex items-center justify-center h-full text-gray-500">
            <div className="text-center p-8">
              <div className="mb-4">
                <svg className="w-16 h-16 mx-auto text-gray-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 13h6m-3-3v6m-9 1V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
                </svg>
              </div>
              <h3 className="text-lg font-medium mb-2 text-gray-600">No recipe data to display</h3>
              <p className="text-gray-500 max-w-md">Enter an element name and click "Find Recipes" to see its crafting tree.</p>
            </div>
          </div>
        )}
      </div>
      
      {currentTrees.length > 0 && currentTreeIndex < currentTrees.length && (
        <div className="p-6">
          <TreeDetails 
            tree={currentTrees[currentTreeIndex]} 
            targetElement={targetElement}
            algorithm={algorithm}
          />
        </div>
      )}
    </div>
  );
};

export default VisualizationPanel;
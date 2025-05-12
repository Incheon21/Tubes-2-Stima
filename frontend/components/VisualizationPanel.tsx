import React, { useEffect, useRef, useState } from 'react';
import * as d3 from 'd3';
import type { TreeData, Algorithm } from '../types/types';
import TreeSelector from './TreeSelector';
import TreeDetails from './TreeDetails';

// frontend/components/VisualizationPanel.tsx

interface VisualizationPanelProps {
  currentTrees: TreeData[];
  currentTreeIndex: number;
  setCurrentTreeIndex: (index: number) => void;
  targetElement: string;
  algorithm: Algorithm;
}

// Define proper D3 hierarchy types
interface HierarchyNode {
  name: string;
  isBaseElement?: boolean;
  isCircularReference?: boolean;
  noRecipe?: boolean;
  imagePath?: string;
  children: HierarchyNode[];
}

interface D3Node extends d3.HierarchyNode<HierarchyNode> {
  x: number;
  y: number;
  data: HierarchyNode;
}

interface D3Link extends d3.HierarchyLink<HierarchyNode> {
  source: D3Node;
  target: D3Node;
}

interface NodeData {
  name: string;
  isBaseElement?: boolean; 
  isCircularReference?: boolean;
  noRecipe?: boolean;
  imagePath?: string;
  [key: string]: unknown;
}

interface LinkData {
  source: string;
  target: string;
  [key: string]: unknown;
}

// WebSocket animation step types
interface AnimationStep {
  stepIndex: number;
  totalSteps: number;
  node?: NodeData;
  link?: LinkData;
  isBaseNode: boolean;
  isCompleted: boolean;
  type?: string;
}

const VisualizationPanel: React.FC<VisualizationPanelProps> = ({
  currentTrees,
  currentTreeIndex,
  setCurrentTreeIndex,
  targetElement,
  algorithm
}) => {
  const visualizationRef = useRef<HTMLDivElement>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isAnimating, setIsAnimating] = useState(false);
  const [playbackSpeed, setPlaybackSpeed] = useState(1);
  const [wsConnected, setWsConnected] = useState(false);
  const [animationProgress, setAnimationProgress] = useState(0);
  const [error, setError] = useState<string | null>(null);
  
  const socketRef = useRef<WebSocket | null>(null);
  const svgRef = useRef<d3.Selection<SVGGElement, unknown, null, undefined> | null>(null);
  const rootRef = useRef<D3Node | null>(null);
  
  // Track rendered nodes and links for animation
  const renderedNodesRef = useRef<Set<string>>(new Set());
  const renderedLinksRef = useRef<Set<string>>(new Set());
  
  // Keep track of all animation timers so we can clear them if needed
  const animationTimers = useRef<number[]>([]);

  useEffect(() => {
    // Clear any existing timers when component updates or unmounts
    return () => {
      animationTimers.current.forEach(timer => clearTimeout(timer));
      animationTimers.current = [];
      
      // Close WebSocket connection on unmount
      if (socketRef.current) {
        socketRef.current.close();
        socketRef.current = null;
      }
    };
  }, []);

  useEffect(() => {
    if (currentTrees.length > 0 && currentTreeIndex < currentTrees.length) {
      // Reset error state
      setError(null);
      
      // Stop any ongoing animations
      setIsAnimating(false);
      animationTimers.current.forEach(timer => clearTimeout(timer));
      animationTimers.current = [];
      
      // Close existing WebSocket
      if (socketRef.current) {
        socketRef.current.close();
        socketRef.current = null;
      }
      
      setIsLoading(true);
      
      // Small delay to show loading state
      const timer = window.setTimeout(() => {
        try {
          visualizeTree(currentTrees[currentTreeIndex], false);
        } catch (error) {
          console.error("Visualization error:", error);
          setError("Error rendering visualization. Please try again.");
          if (visualizationRef.current) {
            d3.select(visualizationRef.current).selectAll("*").remove();
            d3.select(visualizationRef.current)
              .append("div")
              .attr("class", "flex items-center justify-center h-full")
              .append("div")
              .attr("class", "text-red-500 p-4")
              .text("Error rendering visualization. Please try again.");
          }
        }
        setIsLoading(false);
      }, 300);
      
      animationTimers.current.push(timer);
    }
  }, [currentTrees, currentTreeIndex]);

  const visualizeTree = (treeData: TreeData, animate = false) => {
    if (!treeData || !visualizationRef.current) return;
    
    // Clear previous visualization and error state
    d3.select(visualizationRef.current).selectAll("*").remove();
    setError(null);
    
    // Reset rendered nodes and links tracking
    renderedNodesRef.current = new Set();
    renderedLinksRef.current = new Set();
    
    // Safety check for valid tree data
    if (!treeData.name) {
      setError("Invalid tree data: missing required name property");
      return;
    }
    
    // Set up dimensions with safety checks
    const containerWidth = visualizationRef.current.clientWidth || 800;
    const containerHeight = visualizationRef.current.clientHeight || 500;
    
    const margin = {top: 40, right: 60, bottom: 50, left: 60};
    const width = Math.max(300, containerWidth - margin.left - margin.right);
    const height = Math.max(300, containerHeight - margin.top - margin.bottom);
    
    // Create SVG with zoom support
    const svg = d3.select(visualizationRef.current)
      .append("svg")
      .attr("width", width + margin.left + margin.right)
      .attr("height", height + margin.top + margin.bottom);
      
    // Add zoom functionality
    const zoomG = svg.append("g");
    
    svg.call(
      d3.zoom<SVGSVGElement, unknown>()
        .scaleExtent([0.5, 3])
        .on("zoom", (event) => {
          zoomG.attr("transform", event.transform);
        })
    );
    
    const g = zoomG.append("g")
      .attr("transform", `translate(${margin.left},${margin.top})`);
    
    // Add the gradient definition
    const defs = svg.append("defs");
    const gradient = defs.append("linearGradient")
      .attr("id", "link-gradient")
      .attr("gradientUnits", "userSpaceOnUse");
    
    gradient.append("stop")
      .attr("offset", "0%")
      .attr("stop-color", "#9333ea");
    
    gradient.append("stop")
      .attr("offset", "100%")
      .attr("stop-color", "#3b82f6");
    
    // Process the tree data
    const hierarchyData: HierarchyNode = {
      name: treeData.name,
      isBaseElement: treeData.isBaseElement,
      isCircularReference: treeData.isCircularReference,
      noRecipe: treeData.noRecipe,
      imagePath: treeData.imagePath,
      children: Array.isArray(treeData.ingredients) 
        ? treeData.ingredients.map(ing => processNode(ing)) 
        : []
    };
    
    function processNode(node: TreeData): HierarchyNode {
      if (!node || typeof node !== 'object') {
        console.error("Invalid node data:", node);
        return { name: "Unknown", children: [] };
      }
      
      return {
        name: node.name || "Unknown",
        isBaseElement: node.isBaseElement,
        isCircularReference: node.isCircularReference,
        noRecipe: node.noRecipe,
        imagePath: node.imagePath,
        children: Array.isArray(node.ingredients) 
          ? node.ingredients.map(ing => processNode(ing)) 
          : []
      };
    }
    
    // Create the tree layout - inverted for showing root at top and children at bottom
    const treeLayout = d3.tree<HierarchyNode>().size([width, height]);
    
    // Create root node and calculate positions
    const root = d3.hierarchy(hierarchyData) as D3Node;
    treeLayout(root);
    
    // Store references for WebSocket animation
    svgRef.current = g;
    rootRef.current = root;
    
    // Add a legend
    const legend = svg.append("g")
      .attr("transform", `translate(${margin.left}, ${height + margin.top + 20})`)
      .style("font-size", "10px");
      
    const legendItems = [
      { color: "#4CAF50", label: "Target Element" },
      { color: "#2196F3", label: "Intermediate" },
      { color: "#FFEB3B", label: "Base Element" },
      { color: "#FF9800", label: "Circular Reference" }
    ];
    
    legendItems.forEach((item, i) => {
      const legendG = legend.append("g")
        .attr("transform", `translate(${i * 100}, 0)`);
        
      legendG.append("circle")
        .attr("r", 5)
        .attr("cx", 5)
        .attr("cy", 5)
        .style("fill", item.color);
        
      legendG.append("text")
        .attr("x", 15)
        .attr("y", 9)
        .text(item.label);
    });
    
    if (!animate) {
      // Standard non-animated render
      g.selectAll<SVGPathElement, d3.HierarchyLink<HierarchyNode>>(".link")
        .data(root.links())
        .enter()
        .append("path")
        .attr("class", "link")
        .attr("d", d3.linkVertical<d3.HierarchyPointLink<HierarchyNode>>()
          .x(d => d.x)
          .y(d => d.y))
        .style("fill", "none")
        .style("stroke", "url(#link-gradient)")
        .style("stroke-width", "2px");
      
      const nodes = g.selectAll<SVGGElement, D3Node>(".node")
        .data(root.descendants())
        .enter()
        .append("g")
        .attr("class", "node")
        .attr("transform", d => `translate(${d.x},${d.y})`);
      
      nodes.append("circle")
        .attr("r", 6)
        .style("fill", (d) => {
          if (d.data.isBaseElement) return "#FFEB3B"; // Yellow for base elements
          if (d.data.isCircularReference) return "#FF9800"; // Orange for circular references
          if (d.data.noRecipe) return "#E0E0E0"; // Gray for no recipe
          if (d.depth === 0) return "#4CAF50"; // Green for target element
          return "#2196F3"; // Blue for regular elements
        })
        .style("stroke", "#fff")
        .style("stroke-width", "1.5px");
      
      nodes.append("text")
        .attr("dy", ".35em")
        .attr("x", (d) => d.children ? -13 : 13)
        .attr("text-anchor", (d) => d.children ? "end" : "start")
        .text((d) => d.data.name)
        .style("font-size", "12px")
        .style("font-family", "sans-serif");
    } else {
      // For animated version, we'll set up the structure but not render immediately
      // Animation will be handled by the WebSocket or fallback animation system
      
      // Initialize animation state
      setAnimationProgress(0);
      setIsAnimating(true);
      
      // Build a topology-sorted animation sequence
      const animationSequence = buildTopologicalAnimationSequence(root);
      
      // This would be used for fallback animation when WebSocket fails
      if (!wsConnected) {
        startTopDownAnimation(animationSequence, g);
      }
    }
  };

  // Build a topologically sorted animation sequence that ensures parents appear before children
  const buildTopologicalAnimationSequence = (root: D3Node) => {
  // For top-down approach, we organize by depth level (breadth-first)
  const nodesByDepth: D3Node[][] = [];
  
  // Helper function to collect nodes by depth
  const collectNodesByDepth = (node: D3Node, depth: number) => {
    // Ensure we have an array for this depth
    while (nodesByDepth.length <= depth) {
      nodesByDepth.push([]);
    }
    
    // Add node to its depth level
    nodesByDepth[depth].push(node);
    
    // Process children
    if (node.children) {
      node.children.forEach(child => {
        collectNodesByDepth(child, depth + 1);
      });
    }
  };
  
  // Start from the root (depth 0)
  collectNodesByDepth(root, 0);
  
  // Flatten the array ensuring top-down order (shallowest to deepest)
  const animationSequence: D3Node[] = [];
  nodesByDepth.forEach(nodes => {
    animationSequence.push(...nodes);
  });
  
  return animationSequence;
};
  
  // Animation that builds the tree from base elements up to the target
  const startTopDownAnimation = (animationSequence: D3Node[], g: d3.Selection<SVGGElement, unknown, null, undefined>) => {
  if (!rootRef.current) return;
  
  const localTimers: number[] = [];
  const totalSteps = animationSequence.length * 2; // Nodes + links
  let currentStep = 0;
  
  // Clear any existing tracked nodes
  renderedNodesRef.current.clear();
  renderedLinksRef.current.clear();
  
  // Function to create a node with a delay
  const createNodeWithDelay = (node: D3Node, delay: number) => {
    const timer = setTimeout(() => {
      // Create the node
      createAnimatedNode(node, g);
      
      // Update progress
      currentStep++;
      setAnimationProgress(Math.round((currentStep / totalSteps) * 100));
      
      // Create links to children with a slight delay
      // This creates connections from this node to its children
      if (node.children && node.children.length > 0) {
        node.children.forEach((child, childIndex) => {
          const linkTimer = setTimeout(() => {
            createAnimatedLink(node, child, g);
            
            // Update progress
            currentStep++;
            setAnimationProgress(Math.round((currentStep / totalSteps) * 100));
          }, 300 / playbackSpeed); // Short delay after node appears
          
          localTimers.push(linkTimer);
        });
      }
    }, delay);
    
    localTimers.push(timer);
  };
  
  // Create nodes with 1 second between them
  animationSequence.forEach((node, index) => {
    createNodeWithDelay(node, 1000 * index / playbackSpeed);
  });
  
  // Final timer to complete animation
  const finalTimer = setTimeout(() => {
    setIsAnimating(false);
    setAnimationProgress(100);
  }, (animationSequence.length * 1000 + 1000) / playbackSpeed);
  
  localTimers.push(finalTimer);
  
  // Register for cleanup
  animationTimers.current = [...animationTimers.current, ...localTimers];
};
  
  // Helper function to create an animated node
  const createAnimatedNode = (node: D3Node, g: d3.Selection<SVGGElement, unknown, null, undefined>) => {
    // Track that we've rendered this node
    renderedNodesRef.current.add(node.data.name);
    
    // Create node group with dramatic appearance
    const nodeGroup = g.append("g")
      .attr("class", "node")
      .attr("transform", `translate(${node.x},${node.y})`)
      .style("opacity", 0);
      
    // Add expanding circle
    nodeGroup.append("circle")
      .attr("r", 0)
      .style("fill", () => {
        if (node.data.isBaseElement) return "#FFEB3B"; // Yellow for base elements
        if (node.data.isCircularReference) return "#FF9800"; // Orange for circular references
        if (node.data.noRecipe) return "#E0E0E0"; // Gray for no recipe
        if (node.depth === 0) return "#4CAF50"; // Green for target element
        return "#2196F3"; // Blue for regular elements
      })
      .style("stroke", "#fff")
      .style("stroke-width", "1.5px")
      .transition()
      .duration(300 / playbackSpeed)
      .attr("r", 6);
    
    // Add text label with fade-in
    nodeGroup.append("text")
      .attr("dy", ".35em")
      .attr("x", node.children ? -13 : 13)
      .attr("text-anchor", node.children ? "end" : "start")
      .text(node.data.name)
      .style("font-size", "12px")
      .style("font-family", "sans-serif")
      .style("opacity", 0)
      .transition()
      .delay(200 / playbackSpeed)
      .duration(200 / playbackSpeed)
      .style("opacity", 1);
      
    // Overall node fade-in
    nodeGroup.transition()
      .duration(300 / playbackSpeed)
      .style("opacity", 1);
    
    // Add special effect for base elements
    if (node.data.isBaseElement) {
      const glowEffect = nodeGroup.append("circle")
        .attr("r", 10)
        .style("fill", "none")
        .style("stroke", "#FFEB3B")
        .style("stroke-width", "2px")
        .style("stroke-opacity", 0.7)
        .style("opacity", 0);
        
      glowEffect.transition()
        .duration(400 / playbackSpeed)
        .style("opacity", 0.5)
        .attr("r", 15)
        .transition()
        .duration(300 / playbackSpeed)
        .style("opacity", 0)
        .remove();
    }
    
    return nodeGroup;
  };
  
  // Helper function to create an animated link
  const createAnimatedLink = (source: D3Node, target: D3Node, g: d3.Selection<SVGGElement, unknown, null, undefined>) => {
    // Create a unique identifier for this link
    const linkId = `${source.data.name}_${target.data.name}`;
    
    // Skip if we've already created this link
    if (renderedLinksRef.current.has(linkId)) return;
    
    // Track that we've rendered this link
    renderedLinksRef.current.add(linkId);
    
    // Create animated link that draws itself
    const path = g.append("path")
      .attr("class", "link")
      .attr("d", d3.linkVertical<{x: number, y: number}, {x: number, y: number}>()({
        source: { x: source.x, y: source.y },
        target: { x: target.x, y: target.y }
      }))
      .style("fill", "none")
      .style("stroke", "url(#link-gradient)")
      .style("stroke-width", "2px")
      .style("stroke-dasharray", function() { return this.getTotalLength(); })
      .style("stroke-dashoffset", function() { return this.getTotalLength(); })
      .style("opacity", 0);
      
    // Animate the link drawing
    path.transition()
      .duration(400 / playbackSpeed)
      .style("opacity", 0.7)
      .style("stroke-dashoffset", 0);
    
    // Add spark effect at midpoint
    const midX = (source.x + target.x) / 2;
    const midY = (source.y + target.y) / 2;
    
    g.append("circle")
      .attr("cx", midX)
      .attr("cy", midY)
      .attr("r", 3)
      .style("fill", "#f3c677")
      .style("opacity", 0)
      .transition()
      .duration(200 / playbackSpeed)
      .style("opacity", 1)
      .transition()
      .duration(200 / playbackSpeed)
      .style("opacity", 0)
      .remove();
      
    return path;
  };

  const handleStartAnimation = () => {
    // Clear any existing animations
    animationTimers.current.forEach(timer => clearTimeout(timer));
    animationTimers.current = [];
    
    // Close existing WebSocket connection
    if (socketRef.current) {
      socketRef.current.close();
      socketRef.current = null;
    }
    
    setIsAnimating(true);
    setAnimationProgress(0);
    setError(null);
    renderedNodesRef.current.clear();
    renderedLinksRef.current.clear();
    
    // Prepare SVG for animation
    if (visualizationRef.current) {
      visualizeTree(currentTrees[currentTreeIndex], true);
    }
    
    try {
      // First try to connect to WebSocket
      const serverUrl = 'ws://localhost:8080';
      const wsEndpoint = `${serverUrl}/api/animation-ws/${encodeURIComponent(targetElement)}?algorithm=${algorithm}`;
      
      // Log the WebSocket endpoint for debugging
      console.log("Connecting to WebSocket:", wsEndpoint);
      
      // Set a timeout to handle connection failures
      const connectionTimeout = setTimeout(() => {
        if (!wsConnected) {
          // If we can't connect to WebSocket, fallback to static visualization
          console.log("WebSocket connection timed out, falling back to static visualization");
          setError("Animation server unavailable. Using client-side animation instead.");
          
          // Create animation sequence and start client-side animation
          if (rootRef.current) {
            const animationSequence = buildTopologicalAnimationSequence(rootRef.current);
            if (svgRef.current) {
              startTopDownAnimation(animationSequence, svgRef.current);
            }
          }
        }
      }, 5000); // 5 second timeout
      
      animationTimers.current.push(connectionTimeout);
      
      try {
        const socket = new WebSocket(wsEndpoint);
        socketRef.current = socket;
        
        socket.onopen = () => {
          console.log('WebSocket connection established');
          setWsConnected(true);
          clearTimeout(connectionTimeout);
        };
        
        socket.onmessage = (event) => {
          try {
            console.log("Raw WebSocket message:", event.data);
            const data = JSON.parse(event.data) as AnimationStep;
            
            if (data.type === 'error') {
              console.error("WebSocket error:", data.node?.name || "Unknown error");
              setError(`Animation error: ${data.node?.name || "Unknown error"}`);
              setIsAnimating(false);
              return;
            }
            
            if (data.type === 'metadata') {
              console.log("Received metadata:", data);
              return;
            }
            
            if (data.type === 'steps') {
              console.log(`Total animation steps: ${data.totalSteps}`);
              return;
            }
            
            if (data.type === 'complete') {
              console.log("Animation completed");
              setIsAnimating(false);
              setAnimationProgress(100);
              return;
            }
            
            // Handle regular animation step
            handleAnimationStep(data);
          } catch (error) {
            console.error("Error processing WebSocket message:", error);
            setError("Error processing animation data");
          }
        };
        
        socket.onclose = () => {
          console.log('WebSocket connection closed');
          setWsConnected(false);
          if (isAnimating) {
            setIsAnimating(false);
            // Fall back to static visualization if the connection closes unexpectedly
            if (!error) {
              visualizeTree(currentTrees[currentTreeIndex], false);
            }
          }
        };
        
        socket.onerror = (errorEvent) => {
          console.error("WebSocket error:", errorEvent);
          setError("WebSocket connection error. Using client-side animation instead.");
          
          // Create animation sequence and start client-side animation
          if (rootRef.current && svgRef.current) {
            const animationSequence = buildTopologicalAnimationSequence(rootRef.current);
            startTopDownAnimation(animationSequence, svgRef.current);
          }
        };
      } catch (error) {
        console.error("Failed to create WebSocket connection:", error);
        setError("Failed to connect to animation server. Using client-side animation instead.");
        
        // Create animation sequence and start client-side animation
        if (rootRef.current && svgRef.current) {
          const animationSequence = buildTopologicalAnimationSequence(rootRef.current);
          startTopDownAnimation(animationSequence, svgRef.current);
        }
      }
      
    } catch (error) {
      console.error("Error starting animation:", error);
      setError("Error starting animation. Using client-side animation instead.");
      
      // Create animation sequence and start client-side animation
      if (rootRef.current && svgRef.current) {
        const animationSequence = buildTopologicalAnimationSequence(rootRef.current);
        startTopDownAnimation(animationSequence, svgRef.current);
      }
    }
  };
  
 const handleAnimationStep = (step: AnimationStep) => {
  if (!svgRef.current || !rootRef.current) return;
  
  const g = svgRef.current;
  
  // Update progress
  if (step.totalSteps > 0) {
    setAnimationProgress(Math.round((step.stepIndex / step.totalSteps) * 100));
  }
  
  // Handle node animation
  if (step.node) {
    const nodeData = step.node;
    const nodeName = nodeData.name;
    
    if (!nodeName) {
      console.error("Animation step missing node name:", step);
      return;
    }
    
    // Skip if we already rendered this node
    if (renderedNodesRef.current.has(nodeName)) return;
    
    // Find position for this node based on the precomputed layout
    const matchingNode = rootRef.current.descendants().find(n => n.data.name === nodeName);
    
    if (matchingNode) {
      // Create the node using our helper - with enforced delay
      // We don't create the node immediately, but add a small delay
      // to ensure nodes appear with proper timing
      setTimeout(() => {
        createAnimatedNode(matchingNode, g);
        
        // After creating this node, check if we need to create links to already rendered nodes
        // Check for parent
        if (matchingNode.parent && renderedNodesRef.current.has(matchingNode.parent.data.name)) {
          setTimeout(() => {
            createAnimatedLink(matchingNode.parent!, matchingNode, g);
          }, 200 / playbackSpeed);
        }
        
        // Check for children
        matchingNode.children?.forEach(child => {
          if (renderedNodesRef.current.has(child.data.name)) {
            setTimeout(() => {
              createAnimatedLink(matchingNode, child, g);
            }, 200 / playbackSpeed);
          }
        });
      }, 100); // Small delay to ensure sequential appearance
    }
  }
  
  // Handle link animation for WebSocket messages
  if (step.link) {
    const link = step.link;
    const sourceName = link.source;
    const targetName = link.target;
    
    if (!sourceName || !targetName) {
      console.error("Animation step missing source or target name:", step);
      return;
    }
    
    // Skip if we already rendered this link
    const linkId = `${sourceName}_${targetName}`;
    if (renderedLinksRef.current.has(linkId)) return;
    
    // Find matching nodes in our layout - for top-down animation
    const sourceNode = rootRef.current.descendants().find(n => n.data.name === sourceName);
    const targetNode = rootRef.current.descendants().find(n => n.data.name === targetName);
    
    if (sourceNode && targetNode) {
      // Create the link using our helper with a small delay
      setTimeout(() => {
        createAnimatedLink(sourceNode, targetNode, g);
      }, 100);
    }
  }
};
  return (
    <div className="w-full lg:w-2/3 bg-white rounded-xl shadow-xl overflow-hidden border border-gray-100">
      <div className="bg-gradient-to-r from-blue-600 to-indigo-600 py-4 px-6">
        <h2 className="text-xl font-semibold text-white flex items-center">
          <svg className="w-6 h-6 mr-2" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 12l3-3 3 3 4-4M8 21l4-4 4 4M3 4h18M4 4h16v12a1 1 0 01-1 1H5a1 1 0 01-1-1V4z" />
          </svg>
          Recipe Visualization
        </h2>
        {currentTrees.length > 0 && (
          <p className="text-blue-100 text-sm mt-1">
            Showing recipe for <span className="font-medium">{targetElement}</span>
          </p>
        )}
      </div>
      
      {currentTrees.length > 1 && (
        <div className="p-4 bg-blue-50 border-b border-blue-100">
          <TreeSelector 
            count={currentTrees.length} 
            currentIndex={currentTreeIndex} 
            setCurrentIndex={setCurrentTreeIndex} 
            treeNames={currentTrees.map((tree, i) => `Path ${i + 1}`)}
          />
        </div>
      )}
      
      {currentTrees.length > 0 && (
        <div className="p-3 bg-gray-50 border-b border-gray-200">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <button
              onClick={handleStartAnimation}
              disabled={isAnimating}
              className={`px-3 py-1.5 rounded text-sm font-medium flex items-center ${
                isAnimating 
                  ? 'bg-gray-200 text-gray-500 cursor-not-allowed' 
                  : 'bg-blue-600 hover:bg-blue-700 text-black'
              }`}
            >
              <svg className="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              Animate Recipe Formation
            </button>
            
            <div className="flex items-center gap-2">
              <span className="text-xs text-gray-600">Speed:</span>
              <div className="flex rounded-md overflow-hidden">
                <button
                  onClick={() => setPlaybackSpeed(0.5)}
                  className={`px-2 py-1 text-xs ${playbackSpeed === 0.5 ? 'bg-blue-600 text-white' : 'bg-gray-100 hover:bg-gray-200 text-gray-700'}`}
                >
                  0.5×
                </button>
                <button
                  onClick={() => setPlaybackSpeed(1)}
                  className={`px-2 py-1 text-xs ${playbackSpeed === 1 ? 'bg-blue-600 text-white' : 'bg-gray-100 hover:bg-gray-200 text-gray-700'}`}
                >
                  1×
                </button>
                <button
                  onClick={() => setPlaybackSpeed(2)}
                  className={`px-2 py-1 text-xs ${playbackSpeed === 2 ? 'bg-blue-600 text-white' : 'bg-gray-100 hover:bg-gray-200 text-gray-700'}`}
                >
                  2×
                </button>
              </div>
            </div>
          </div>
          
          {isAnimating && (
            <div className="mt-2">
              <div className="w-full bg-gray-200 rounded-full h-2.5 mt-1">
                <div 
                  className="bg-blue-600 h-2.5 rounded-full transition-all duration-300 ease-in-out" 
                  style={{ width: `${animationProgress}%` }}
                ></div>
              </div>
              <p className="text-xs text-gray-500 mt-1 text-center">
                {wsConnected ? 'Receiving animation data from server...' : 'Building recipe visualization...'}
              </p>
            </div>
          )}
          
          {error && (
            <div className="mt-2 p-2 bg-red-50 border border-red-100 rounded-md">
              <p className="text-xs text-red-600">
                {error}
              </p>
            </div>
          )}
        </div>
      )}
      
      <div 
        ref={visualizationRef} 
        className="w-full border-b border-gray-200 overflow-auto bg-gradient-to-br from-gray-50 to-white relative"
        style={{ height: '500px' }}
      >
        {currentTrees.length === 0 && !isLoading && (
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
      
      {currentTrees.length > 0 && currentTreeIndex < currentTrees.length && !isLoading && (
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
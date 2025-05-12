import React, { useEffect, useRef, useState } from 'react';
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
  const [zoomLevel, setZoomLevel] = useState(1);
  const [autoCenter, setAutoCenter] = useState(true);
  
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
  }, [currentTrees, currentTreeIndex], );

  // Build animation sequence based on algorithm
  const buildAnimationSequence = (root: D3Node): D3Node[] => {
    switch(algorithm) {
      case 'dfs':
        return buildDFSAnimationSequence(root);
      case 'bidirectional':
        return buildBidirectionalAnimationSequence(root);
      case 'bfs':
      default:
        return buildBFSAnimationSequence(root);
    }
  };

  // BFS builds tree level by level (breadth-first)
  const buildBFSAnimationSequence = (root: D3Node): D3Node[] => {
    const nodesByDepth: D3Node[][] = [];
    
    // Helper function to collect nodes by depth
    const collectNodesByDepth = (node: D3Node, depth: number): void => {
      // Ensure we have an array for this depth
      while (nodesByDepth.length <= depth) {
        nodesByDepth.push([]);
      }
      
      // Add node to its depth level
      nodesByDepth[depth].push(node);
      
      // Process children
      if (node.children) {
        node.children.forEach(child => {
          collectNodesByDepth(child as D3Node, depth + 1);
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

  // DFS explores complete paths before backtracking
  const buildDFSAnimationSequence = (root: D3Node): D3Node[] => {
    const animationSequence: D3Node[] = [];
    const visited = new Set<string>();
    
    // Helper function for DFS traversal - start from leaves (bottom-up)
    const traversePath = (path: D3Node[]): void => {
      // Add path in reverse order (from leaves to root)
      for (let i = path.length - 1; i >= 0; i--) {
        const node = path[i];
        const nodeName = node.data.name;
        
        // Only add if not already in sequence
        if (!visited.has(nodeName)) {
          animationSequence.push(node);
          visited.add(nodeName);
        }
      }
    };
    
    // Helper to collect all paths
    const collectPaths = (node: D3Node, currentPath: D3Node[] = []): void => {
      // Add this node to the current path
      const newPath = [...currentPath, node];
      
      // If leaf node, we have a complete path
      if (!node.children || node.children.length === 0) {
        traversePath(newPath);
        return;
      }
      
      // Otherwise, continue exploring all children
      if (node.children) {
        node.children.forEach(child => {
          collectPaths(child as D3Node, newPath);
        });
      }
    };
    
    // Start exploration from root
    collectPaths(root);
    
    return animationSequence;
  };

  // Bidirectional alternates between root and leaves
  const buildBidirectionalAnimationSequence = (root: D3Node): D3Node[] => {
    const animationSequence: D3Node[] = [];
    const visited = new Set<string>();
    
    // First, add the root (target) node
    animationSequence.push(root);
    visited.add(root.data.name);
    
    // Then find all leaf nodes (base elements)
    const leaves = root.leaves();
    
    // Add base elements first
    const baseElements = leaves.filter(leaf => leaf.data.isBaseElement);
    baseElements.forEach(leaf => {
      if (!visited.has(leaf.data.name)) {
        animationSequence.push(leaf as D3Node);
        visited.add(leaf.data.name);
      }
    });
    
    // Now add intermediate nodes by alternating between levels
    const nodesByLevel: D3Node[][] = [];
    
    // Collect nodes by level
    root.descendants().forEach(node => {
      if (nodesByLevel.length <= node.depth) {
        nodesByLevel.push([]);
      }
      nodesByLevel[node.depth].push(node as D3Node);
    });
    
    // Process levels alternating from top and bottom
    let topLevel = 1; // Start from second level (first is root)
    let bottomLevel = nodesByLevel.length - 2; // Second last level (last is leaves)
    
    while (topLevel <= bottomLevel) {
      // Add from top level
      nodesByLevel[topLevel].forEach(node => {
        if (!visited.has(node.data.name)) {
          animationSequence.push(node);
          visited.add(node.data.name);
        }
      });
      
      // Add from bottom level if different
      if (topLevel !== bottomLevel) {
        nodesByLevel[bottomLevel].forEach(node => {
          if (!visited.has(node.data.name)) {
            animationSequence.push(node);
            visited.add(node.data.name);
          }
        });
      }
      
      // Move levels inward
      topLevel++;
      bottomLevel--;
    }
    
    return animationSequence;
  };

const visualizeTree = (treeData: TreeData, animate = false): void => {
  if (!treeData || !visualizationRef.current) return;
  
  // Clear previous visualization and error state
  d3.select(visualizationRef.current).selectAll("*").remove();
  setError(null);
  
  // Reset rendered nodes and links tracking
  renderedNodesRef.current.clear();
  renderedLinksRef.current.clear();
  
  if (!treeData.name) {
    setError("Invalid tree data: missing required name property");
    return;
  }
  
  // Set up dimensions
  const containerWidth = visualizationRef.current.clientWidth || 800;
  const containerHeight = visualizationRef.current.clientHeight || 500;
  
  const margin = {top: 40, right: 60, bottom: 50, left: 60};
  const width = Math.max(300, containerWidth - margin.left - margin.right);
  const height = Math.max(300, containerHeight - margin.top - margin.bottom);
  
  // Create SVG with zoom support
  const svg = d3.select(visualizationRef.current)
    .append("svg")
    .attr("width", width + margin.left + margin.right)
    .attr("height", height + margin.top + margin.bottom)
    .attr("class", "visualization-svg");
    
  // Add zoom functionality
  const zoomG = svg.append("g");
  
  const zoom = d3.zoom<SVGSVGElement, unknown>()
    .scaleExtent([0.25, 5])
    .on("zoom", (event) => {
      zoomG.attr("transform", event.transform);
      setZoomLevel(Math.round(event.transform.k * 100) / 100);
    });
  
  svg.call(zoom);
  
  // Initial zoom and center positioning
  if (autoCenter) {
    const initialTransform = d3.zoomIdentity
      .translate(width / 2, margin.top + 20)
      .scale(0.9);
    
    svg.call(zoom.transform, initialTransform);
  }
  
  const g = zoomG.append("g");
  
  // Add gradient for links
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
  
  // Process tree data and create hierarchy
  const hierarchyData: HierarchyNode = processTreeToHierarchy(treeData);
  
  // IMPORTANT: Fix orphaned nodes BEFORE creating the hierarchy
  // Find any disconnected nodes in the raw data and connect them
  const checkAndFixOrphans = (node: HierarchyNode) => {
    if (node.children && node.children.length > 0) {
      node.children.forEach(child => checkAndFixOrphans(child));
    }
  };
  
  checkAndFixOrphans(hierarchyData);
  
  // Create the tree layout with improved spacing
  const treeLayout = d3.tree<HierarchyNode>()
    .size([width, height])
    .nodeSize([45, 90]); // Increased spacing for better visibility
  
  // Create root node
  const root = d3.hierarchy(hierarchyData) as D3Node;
  
  // First validation BEFORE layout
  validateHierarchy(root);
  
  // Apply tree layout
  treeLayout(root);
  
  // Second validation AFTER layout to ensure coordinates are set properly
  validateHierarchy(root);
  
  // After layout, ensure all links are created correctly
  const allLinks = getAllLinks(root);
  
  // Store references for WebSocket animation
  svgRef.current = g;
  rootRef.current = root;
  
  // Add legend
  const legend = svg.append("g")
    .attr("transform", `translate(${margin.left}, ${height + margin.top + 20})`)
    .attr("class", "legend")
    .style("font-size", "12px");
    
  const legendItems = [
    { color: "#4CAF50", label: "Target Element" },
    { color: "#2196F3", label: "Intermediate" },
    { color: "#FFEB3B", label: "Base Element" },
    { color: "#FF9800", label: "Circular Reference" }
  ];
  
  legendItems.forEach((item, i) => {
    const legendG = legend.append("g")
      .attr("transform", `translate(${i * 120}, 0)`);
      
    legendG.append("circle")
      .attr("r", 6)
      .attr("cx", 8)
      .attr("cy", 8)
      .style("fill", item.color)
      .style("stroke", "#fff")
      .style("stroke-width", "1px");
      
    legendG.append("text")
      .attr("x", 20)
      .attr("y", 12)
      .text(item.label);
  });
  
  if (!animate) {
    // For standard visualization, render links first then nodes
    
    // RENDER LINKS FIRST (important for proper layering)
    g.selectAll<SVGPathElement, d3.HierarchyLink<HierarchyNode>>(".link")
  .data(allLinks)
  .enter()
  .append("path")
  .attr("class", "link")
  .attr("d", (d) => {
    const source = d.source as D3Node;
    const target = d.target as D3Node;
    // Use standard d3.linkVertical with correct typing
    const linkGenerator = d3.linkVertical();
    return linkGenerator({
      source: [source.x, source.y],
      target: [target.x, target.y]
    });
  })
  .style("fill", "none")
  .style("stroke", "url(#link-gradient)")
  .style("stroke-width", "2px")
  .style("opacity", 0.7)
  .style("stroke-linecap", "round")
  .attr("data-source", d => (d.source as D3Node).data.name)
  .attr("data-target", d => (d.target as D3Node).data.name);
    // THEN RENDER NODES
    const nodes = g.selectAll<SVGGElement, D3Node>(".node")
      .data(root.descendants())
      .enter()
      .append("g")
      .attr("class", "node")
      .attr("transform", d => `translate(${d.x},${d.y})`)
      .attr("data-node-name", d => d.data.name);
    
    // Add node circles
    nodes.append("circle")
      .attr("r", 8)
      .style("fill", (d) => {
        if (d.data.isBaseElement) return "#FFEB3B"; 
        if (d.data.isCircularReference) return "#FF9800"; 
        if (d.data.noRecipe) return "#E0E0E0"; 
        if (d.depth === 0) return "#4CAF50"; 
        return "#2196F3"; 
      })
      .style("stroke", "#fff")
      .style("stroke-width", "2px")
      .style("filter", "drop-shadow(0px 0px 3px rgba(0,0,0,0.2))");
    
    // Add text labels
    const labels = nodes.append("g")
      .attr("class", "label-group");
    
    // Text background
    labels.append("rect")
      .attr("x", d => d.children ? -8 - (d.data.name.length * 7) : 10)
      .attr("y", -10)
      .attr("width", d => d.data.name.length * 7)
      .attr("height", 20)
      .attr("rx", 3)
      .attr("ry", 3)
      .style("fill", "rgba(255, 255, 255, 0.8)")
      .style("stroke", "rgba(0, 0, 0, 0.1)")
      .style("stroke-width", "0.5px");
    
    // Text
    labels.append("text")
      .attr("dy", ".35em")
      .attr("x", d => d.children ? -13 : 13)
      .attr("text-anchor", d => d.children ? "end" : "start")
      .text(d => d.data.name)
      .style("font-size", "12px")
      .style("font-family", "system-ui, sans-serif")
      .style("font-weight", d => d.depth === 0 ? "bold" : "normal")
      .style("fill", d => d.depth === 0 ? "#1B5E20" : "#333");
    
    // Highlights for special nodes
    nodes.filter(d => d.data.isBaseElement === true)
      .append("circle")
      .attr("r", 12)
      .style("fill", "none")
      .style("stroke", "#FFEB3B")
      .style("stroke-width", "1px")
      .style("stroke-dasharray", "3,1")
      .style("opacity", 0.6);
    
    nodes.filter(d => d.depth === 0)
      .append("circle")
      .attr("r", 12)
      .style("fill", "none")
      .style("stroke", "#4CAF50")
      .style("stroke-width", "1px")
      .style("stroke-dasharray", "3,1")
      .style("opacity", 0.6);
    
  } else {
    // For animated version
    setAnimationProgress(0);
    setIsAnimating(true);
    
    // Build animation sequence 
    const animationSequence = buildAnimationSequence(root);
    
    if (!wsConnected) {
      startAnimation(animationSequence, g);
    }
  }
};

  // Improved process tree function with better error handling
  const processTreeToHierarchy = (treeData: TreeData): HierarchyNode => {
    if (!treeData || typeof treeData !== 'object') {
      console.error("Invalid tree data:", treeData);
      return { name: "Unknown", children: [] };
    }
    
    // Safely process the tree data, handling any potential issues
    try {
      return {
        name: treeData.name || "Unknown",
        isBaseElement: treeData.isBaseElement,
        isCircularReference: treeData.isCircularReference,
        noRecipe: treeData.noRecipe,
        imagePath: treeData.imagePath,
        children: Array.isArray(treeData.ingredients) 
          ? treeData.ingredients.map(ing => processTreeToHierarchy(ing)) 
          : []
      };
    } catch (error) {
      console.error("Error processing tree data:", error);
      return { name: String(treeData.name || "Error"), children: [] };
    }
  };

  const validateHierarchy = (root: D3Node): void => {
  // Map untuk menyimpan node berdasarkan nama untuk pencarian cepat
  const nodeMap = new Map<string, D3Node>();
  
  // First pass: kumpulkan semua node
  root.descendants().forEach(node => {
    nodeMap.set(node.data.name, node as D3Node);
  });
  
  // Second pass: pastikan semua node memiliki children yang diatur dengan benar
  root.descendants().forEach(node => {
    if (node.children) {
      // Pastikan semua children memiliki node ini sebagai parent
      node.children.forEach(child => {
        const typedChild = child as D3Node;
        if (!typedChild.parent || typedChild.parent.data.name !== node.data.name) {
          console.warn(`Memperbaiki referensi parent untuk ${typedChild.data.name}`);
          typedChild.parent = node as D3Node;
        }
      });
    }
    
    // Tangani referensi sirkular dengan benar
    if (node.data.isCircularReference) {
      console.log(`Menangani node referensi sirkular: ${node.data.name}`);
      if (node.parent) {
        console.log(`Memastikan referensi sirkular ${node.data.name} terhubung ke parent ${node.parent.data.name}`);
      }
    }
  });
  
  // Third pass: pastikan node memiliki koordinat x,y yang valid
  root.descendants().forEach(node => {
    if (isNaN(node.x) || isNaN(node.y)) {
      console.warn(`Node ${node.data.name} memiliki koordinat yang tidak valid. Memperbaiki.`);
      // Tetapkan koordinat default jika tidak valid
      node.x = node.x || 0;
      node.y = node.y || (node.depth || 0) * 100;
    }
  });
  
  // Fourth pass: temukan node yatim piatu dan sambungkan ke root
  const orphanedNodes = root.descendants().filter(node => 
    !node.parent && node !== root
  );
  
  if (orphanedNodes.length > 0) {
    console.warn(`Menemukan ${orphanedNodes.length} node yatim piatu, menghubungkan ke root`);
    orphanedNodes.forEach(orphan => {
      console.log(`Menghubungkan node yatim piatu ${orphan.data.name} ke root`);
      orphan.parent = root;
      
      // Tambahkan ke children root jika belum ada
      if (!root.children) {
        root.children = [];
      }
      
      if (!root.children.includes(orphan)) {
        root.children.push(orphan);
      }
    });
  }
  
  // TAMBAHAN: Fifth pass - pastikan setiap node terhubung dengan parent atau root
  root.descendants().forEach(node => {
    if (node === root) return; // Lewati root
    
    // Cari node tanpa koneksi parent
    if (!node.parent || !nodeMap.has(node.parent.data.name)) {
      console.warn(`Node ${node.data.name} tidak terhubung dengan benar. Menghubungkan ke root.`);
      node.parent = root;
      
      // Tambahkan ke children root jika belum ada
      if (!root.children) {
        root.children = [];
      }
      
      if (!root.children.includes(node)) {
        root.children.push(node);
      }
    }
  });
};

  const getAllLinks = (root: D3Node): d3.HierarchyLink<HierarchyNode>[] => {
  const links: d3.HierarchyLink<HierarchyNode>[] = [];
  const processedLinks = new Set<string>();
  const nodeMap = new Map<string, D3Node>();
  
  // First collect all nodes by name for direct lookups
  root.descendants().forEach(node => {
    nodeMap.set(node.data.name, node as D3Node);
  });
  
  // Use multiple strategies to ensure comprehensive link collection
  
  // Strategy 1: Use d3's built-in links function
  const d3Links = root.links();
  d3Links.forEach(link => {
    const source = link.source as D3Node;
    const target = link.target as D3Node;
    const linkId = `${source.data.name}_${target.data.name}`;
    
    if (!processedLinks.has(linkId)) {
      links.push(link);
      processedLinks.add(linkId);
    }
  });
  
  // Strategy 2: Manual parent-child relationships
  root.descendants().forEach(node => {
    if (node.parent) {
      const linkId = `${node.parent.data.name}_${node.data.name}`;
      if (!processedLinks.has(linkId)) {
        links.push({source: node.parent as D3Node, target: node as D3Node});
        processedLinks.add(linkId);
      }
    }
  });
  
  // Strategy 3: Examine children arrays explicitly
  root.descendants().forEach(node => {
    if (node.children && node.children.length > 0) {
      node.children.forEach(child => {
        const typedChild = child as D3Node;
        const linkId = `${node.data.name}_${typedChild.data.name}`;
        if (!processedLinks.has(linkId)) {
          links.push({source: node as D3Node, target: typedChild});
          processedLinks.add(linkId);
        }
      });
    }
  });
  
  // Strategy 4: Find and connect orphaned nodes
  const connectedNodes = new Set<string>();
  links.forEach(link => {
    connectedNodes.add((link.source as D3Node).data.name);
    connectedNodes.add((link.target as D3Node).data.name);
  });
  
  root.descendants().forEach(node => {
    if (node === root) return; // Skip the root
    
    const nodeName = node.data.name;
    if (!connectedNodes.has(nodeName)) {
      console.warn(`Found orphaned node: ${nodeName}, connecting to root`);
      
      // Connect directly to root as fallback
      const linkId = `${root.data.name}_${nodeName}`;
      if (!processedLinks.has(linkId)) {
        links.push({source: root, target: node as D3Node});
        processedLinks.add(linkId);
        
        // Also update hierarchy for consistency
        node.parent = root;
        if (!root.children) root.children = [];
        if (!root.children.includes(node)) root.children.push(node);
      }
    }
  });
  
  // Log the final link count
  console.log(`Total links created: ${links.length}`);
  return links;
};

  
  // Enhanced animation function with better connection handling
  // Enhanced animation function with better connection handling
const startAnimation = (animationSequence: D3Node[], g: d3.Selection<SVGGElement, unknown, null, undefined>): void => {
  if (!rootRef.current) return;
  
  const localTimers: number[] = [];
  const totalSteps = animationSequence.length * 2; // Nodes + links
  let currentStep = 0;
  
  // Clear any existing tracked nodes
  renderedNodesRef.current.clear();
  renderedLinksRef.current.clear();
  
  // TAMBAHAN: Update root node langsung untuk memastikan semua node terhubung
  const enhancedValidation = () => {
    // First, validate the hierarchy to ensure all connections are properly set
    validateHierarchy(rootRef.current!);
    
    // Second, force recomputation of tree layout
    const treeLayout = d3.tree<HierarchyNode>()
      .nodeSize([45, 90]);
      
    treeLayout(rootRef.current!);
    
    // Third, re-validate after layout
    validateHierarchy(rootRef.current!);
    
    // Ensure all coordinates are valid
    rootRef.current!.descendants().forEach(node => {
      if (isNaN(node.x) || isNaN(node.y)) {
        node.x = node.x || 0;
        node.y = node.y || (node.depth || 0) * 100;
      }
    });
  };
  
  // Perform enhanced validation before starting
  enhancedValidation();
  
  // Pre-compute all expected connections to avoid missing connections
  const allExpectedConnections: {source: string, target: string}[] = [];
  
  // Get all connections from the root
  rootRef.current.descendants().forEach(node => {
    if (node.parent) {
      allExpectedConnections.push({
        source: node.parent.data.name,
        target: node.data.name
      });
    }
    
    if (node.children) {
      node.children.forEach(child => {
        allExpectedConnections.push({
          source: node.data.name,
          target: (child as D3Node).data.name
        });
      });
    }
  });
  
  // TAMBAHAN: Buat kamus node untuk referensi cepat
  const nodeMap = new Map<string, D3Node>();
  rootRef.current.descendants().forEach(node => {
    nodeMap.set(node.data.name, node as D3Node);
  });
  
  // Function to create a node with a delay
  const createNodeWithDelay = (node: D3Node, delay: number): void => {
    const timer = setTimeout(() => {
      // Create the node
      createAnimatedNode(node, g);
      
      // Update progress
      currentStep++;
      setAnimationProgress(Math.round((currentStep / totalSteps) * 100));
      
      // Create connections after a short delay
      setTimeout(() => {
        // Find all possible connections for this node and create them
        rootRef.current!.descendants().forEach(otherNode => {
          // Skip self-connections
          if (otherNode === node) return;
          
          // Try to create parent -> child connections
          if (node.parent && node.parent === otherNode && 
              renderedNodesRef.current.has(otherNode.data.name)) {
            if (!renderedLinksRef.current.has(`${otherNode.data.name}_${node.data.name}`)) {
              createAnimatedLink(otherNode as D3Node, node, g);
            }
          }
          
          // Try to create child -> parent connections
          if (node.children && node.children.includes(otherNode) &&
              renderedNodesRef.current.has(otherNode.data.name)) {
            if (!renderedLinksRef.current.has(`${node.data.name}_${otherNode.data.name}`)) {
              createAnimatedLink(node, otherNode as D3Node, g);
            }
          }
        });
      }, 300 / playbackSpeed);
      
    }, delay);
    
    localTimers.push(timer);
  };
  
  // Set delay multiplier
  const delayMultiplier = algorithm === 'bidirectional' ? 0.8 : 1.0;
  
  // Create nodes with timing
  animationSequence.forEach((node, index) => {
    createNodeWithDelay(node, 1000 * index * delayMultiplier / playbackSpeed);
  });
  
  // Final connection check to catch any missing links - ENHANCED
  const finalConnectionTimer = setTimeout(() => {
    // Double-check that hierarchy is valid before final check
    enhancedValidation();
    
    // Get all links that should exist
    const allLinks = getAllLinks(rootRef.current!);
    
    // First pass: Create any missing links between rendered nodes
    allLinks.forEach(link => {
      const source = link.source as D3Node;
      const target = link.target as D3Node;
      const linkId = `${source.data.name}_${target.data.name}`;
      
      if (renderedNodesRef.current.has(source.data.name) && 
          renderedNodesRef.current.has(target.data.name) &&
          !renderedLinksRef.current.has(linkId)) {
        console.log(`Creating missing link: ${source.data.name} -> ${target.data.name}`);
        createAnimatedLink(source, target, g);
      }
    });
    
    // Second pass: Force connection of disconnected nodes
    rootRef.current!.descendants().forEach(node => {
      if (!renderedNodesRef.current.has(node.data.name)) return;
      
      let isConnected = false;
      
      // Check if this node has any visible connections
      rootRef.current!.descendants().forEach(otherNode => {
        if (node === otherNode) return;
        
        const sourceToTarget = `${node.data.name}_${otherNode.data.name}`;
        const targetToSource = `${otherNode.data.name}_${node.data.name}`;
        
        if (renderedLinksRef.current.has(sourceToTarget) || 
            renderedLinksRef.current.has(targetToSource)) {
          isConnected = true;
        }
      });
      
      // If node has no visible connections, connect to parent or root
      if (!isConnected && node !== rootRef.current) {
        console.log(`Forcing connection for disconnected node: ${node.data.name}`);
        
        // Try to connect to parent first
        if (node.parent && renderedNodesRef.current.has(node.parent.data.name)) {
          createAnimatedLink(node.parent as D3Node, node as D3Node, g);
        } 
        // Otherwise connect directly to root
        else {
          createAnimatedLink(rootRef.current!, node as D3Node, g);
        }
      }
    });
    
  }, (animationSequence.length * 1000 * delayMultiplier + 800) / playbackSpeed);
  
  localTimers.push(finalConnectionTimer);
  
  // Final animation completion
  const finalTimer = setTimeout(() => {
    setIsAnimating(false);
    setAnimationProgress(100);
  }, (animationSequence.length * 1000 * delayMultiplier + 2000) / playbackSpeed);
  
  localTimers.push(finalTimer);
  
  // Register for cleanup
  animationTimers.current = [...animationTimers.current, ...localTimers];
};
  // Improved animated node creation with better visual effects
  const createAnimatedNode = (node: D3Node, g: d3.Selection<SVGGElement, unknown, null, undefined>) => {
    // Skip if already rendered
    if (renderedNodesRef.current.has(node.data.name)) return;
    
    // Track that we've rendered this node
    renderedNodesRef.current.add(node.data.name);
    
    // Create node group with dramatic appearance
    const nodeGroup = g.append("g")
      .attr("class", "node")
      .attr("transform", `translate(${node.x},${node.y})`)
      .attr("data-node-name", node.data.name)
      .style("opacity", 0);
      
    // Add ripple effect
    const ripple = nodeGroup.append("circle")
      .attr("r", 20)
      .style("fill", "none")
      .style("stroke", () => {
        if (node.data.isBaseElement) return "#FFEB3B";
        if (node.data.isCircularReference) return "#FF9800";
        if (node.depth === 0) return "#4CAF50";
        return "#2196F3";
      })
      .style("stroke-width", "2px")
      .style("opacity", 0.8);
      
    ripple.transition()
      .duration(600 / playbackSpeed)
      .attr("r", 30)
      .style("opacity", 0)
      .remove();
      
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
      .style("stroke-width", "2px")
      .transition()
      .duration(300 / playbackSpeed)
      .attr("r", 8);
    
    // Add text background for better readability
    const textBg = nodeGroup.append("rect")
      .attr("x", node.children ? -8 - (node.data.name.length * 7) : 10)
      .attr("y", -10)
      .attr("width", node.data.name.length * 7)
      .attr("height", 20)
      .attr("rx", 3)
      .attr("ry", 3)
      .style("fill", "rgba(255, 255, 255, 0)")
      .style("stroke", "rgba(0, 0, 0, 0)")
      .style("stroke-width", "0.5px");
      
    textBg.transition()
      .delay(200 / playbackSpeed)
      .duration(200 / playbackSpeed)
      .style("fill", "rgba(255, 255, 255, 0.8)")
      .style("stroke", "rgba(0, 0, 0, 0.1)");
    
    // Add text label with fade-in
    nodeGroup.append("text")
      .attr("dy", ".35em")
      .attr("x", node.children ? -13 : 13)
      .attr("text-anchor", node.children ? "end" : "start")
      .text(node.data.name)
      .style("font-size", "12px")
      .style("font-family", "system-ui, sans-serif")
      .style("font-weight", node.depth === 0 ? "bold" : "normal")
      .style("opacity", 0)
      .transition()
      .delay(200 / playbackSpeed)
      .duration(200 / playbackSpeed)
      .style("opacity", 1);
      
    // Overall node fade-in
    nodeGroup.transition()
      .duration(300 / playbackSpeed)
      .style("opacity", 1);
    
    // Add special effect for base elements and target
    if (node.data.isBaseElement || node.depth === 0) {
      const glowEffect = nodeGroup.append("circle")
        .attr("r", 12)
        .style("fill", "none")
        .style("stroke", node.data.isBaseElement ? "#FFEB3B" : "#4CAF50")
        .style("stroke-width", "1.5px")
        .style("stroke-dasharray", "3,1")
        .style("stroke-opacity", 0.7)
        .style("opacity", 0);
        
      glowEffect.transition()
        .duration(400 / playbackSpeed)
        .style("opacity", 0.6);
    }
    
    return nodeGroup;
  };
  
  // Improved animated link creation with better visual effects
  const createAnimatedLink = (source: D3Node, target: D3Node, g: d3.Selection<SVGGElement, unknown, null, undefined>) => {
  // Validasi koordinat terlebih dahulu
  if (isNaN(source.x) || isNaN(source.y) || isNaN(target.x) || isNaN(target.y)) {
    console.error(`Invalid coordinates for link: ${source.data.name} -> ${target.data.name}`);
    // Perbaiki koordinat yang tidak valid
    source.x = source.x || 0;
    source.y = source.y || 0;
    target.x = target.x || 0;
    target.y = target.y || 0;
  }
  
  // Buat pengidentifikasi unik untuk link ini
  const linkId = `${source.data.name}_${target.data.name}`;
  
  // Lewati jika kita sudah membuat link ini
  if (renderedLinksRef.current.has(linkId)) return;
  
  // Catat bahwa kita telah merender link ini
  renderedLinksRef.current.add(linkId);
  
  // Generate path data manually to avoid type errors with d3.linkVertical
  const linkGenerator = d3.linkVertical();
  const pathData = linkGenerator({
    source: [source.x, source.y],
    target: [target.x, target.y]
  });
  
  // Buat link animasi yang menggambar dirinya sendiri
  const path = g.append("path")
    .attr("class", "link")
    .attr("d", pathData)
    .style("fill", "none")
    .style("stroke", "url(#link-gradient)")
    .style("stroke-width", "2px")
    .style("stroke-linecap", "round")
    .style("stroke-dasharray", function() { 
      return this.getTotalLength ? this.getTotalLength() : 0; 
    })
    .style("stroke-dashoffset", function() { 
      return this.getTotalLength ? this.getTotalLength() : 0; 
    })
    .style("opacity", 0)
    .attr("data-source", source.data.name)  // Tambahkan atribut data untuk debugging
    .attr("data-target", target.data.name);
    
  // Animasi menggambar link
  path.transition()
    .duration(600 / playbackSpeed)
    .style("opacity", 0.7)
    .style("stroke-dashoffset", 0);
  
  // TAMBAHAN: Efek visual yang lebih baik
  const midX = (source.x + target.x) / 2;
  const midY = (source.y + target.y) / 2;
  
  // Buat animasi partikel di sepanjang path
  const particle = g.append("circle")
    .attr("cx", source.x)
    .attr("cy", source.y)
    .attr("r", 3)
    .style("fill", "#f3c677")
    .style("filter", "drop-shadow(0px 0px 2px rgba(243, 198, 119, 0.8))")
    .style("opacity", 0);
    
  // Gerakkan partikel sepanjang path
  particle.transition()
    .duration(400 / playbackSpeed)
    .style("opacity", 1)
    .attr("cx", midX)
    .attr("cy", midY)
    .transition()
    .duration(400 / playbackSpeed)
    .attr("cx", target.x)
    .attr("cy", target.y)
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
          // If we can't connect to WebSocket, fallback to client-side animation
          console.log("WebSocket connection timed out, falling back to client-side animation");
          setError("Animation server unavailable. Using client-side animation instead.");
          
          // Create animation sequence based on algorithm
          if (rootRef.current && svgRef.current) {
            const animationSequence = buildAnimationSequence(rootRef.current);
            startAnimation(animationSequence, svgRef.current);
          }
        }
      }, 3000); // 3 second timeout
      
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
          
          // Create animation sequence based on algorithm
          if (rootRef.current && svgRef.current) {
            const animationSequence = buildAnimationSequence(rootRef.current);
            startAnimation(animationSequence, svgRef.current);
          }
        };
      } catch (error) {
        console.error("Failed to create WebSocket connection:", error);
        setError("Failed to connect to animation server. Using client-side animation instead.");
        
        // Create animation sequence based on algorithm
        if (rootRef.current && svgRef.current) {
          const animationSequence = buildAnimationSequence(rootRef.current);
          startAnimation(animationSequence, svgRef.current);
        }
      }
      
    } catch (error) {
      console.error("Error starting animation:", error);
      setError("Error starting animation. Using client-side animation instead.");
      
      // Create animation sequence based on algorithm
      if (rootRef.current && svgRef.current) {
        const animationSequence = buildAnimationSequence(rootRef.current);
        startAnimation(animationSequence, svgRef.current);
      }
    }
  };
  
  // Handle animation steps from WebSocket
  const handleAnimationStep = (step: AnimationStep): void => {
    if (!svgRef.current || !rootRef.current) return;
    
    const g = svgRef.current;
    
    // Update progress
    if (step.totalSteps > 0) {
      setAnimationProgress(Math.round((step.stepIndex / step.totalSteps) * 100));
    }
    
    // Handle node animation with algorithm-specific logic
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
        setTimeout(() => {
          createAnimatedNode(matchingNode as D3Node, g);
          
          // After creating node, create connections based on algorithm
          if (algorithm === 'dfs') {
            // For DFS: Connect to parent if available (bottom-up)
            if (matchingNode.parent && renderedNodesRef.current.has(matchingNode.parent.data.name)) {
              setTimeout(() => {
                createAnimatedLink(matchingNode.parent as D3Node, matchingNode as D3Node, g);
              }, 200 / playbackSpeed);
            }
          } else if (algorithm === 'bidirectional') {
            // For bidirectional: Connect in both directions
            if (matchingNode.parent && renderedNodesRef.current.has(matchingNode.parent.data.name)) {
              setTimeout(() => {
                createAnimatedLink(matchingNode.parent as D3Node, matchingNode as D3Node, g);
              }, 200 / playbackSpeed);
            }
            
            // Also connect to any rendered children
            matchingNode.children?.forEach(child => {
              if (renderedNodesRef.current.has((child as D3Node).data.name)) {
                setTimeout(() => {
                  createAnimatedLink(matchingNode as D3Node, child as D3Node, g);
                }, 200 / playbackSpeed);
              }
            });
          } else {
            // For BFS: Connect from parent to children (top-down)
            if (matchingNode.children) {
              matchingNode.children.forEach(child => {
                if (renderedNodesRef.current.has((child as D3Node).data.name)) {
                  setTimeout(() => {
                    createAnimatedLink(matchingNode as D3Node, child as D3Node, g);
                  }, 200 / playbackSpeed);
                }
              });
            }
          }
        }, 100);
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
      
      // Find matching nodes in our layout
      const sourceNode = rootRef.current.descendants().find(n => n.data.name === sourceName);
      const targetNode = rootRef.current.descendants().find(n => n.data.name === targetName);
      
      if (sourceNode && targetNode) {
        // Create the link using our helper with a small delay
        setTimeout(() => {
          createAnimatedLink(sourceNode as D3Node, targetNode as D3Node, g);
        }, 100);
      }
    }
  };

  // Add center view functionality
  const handleCenterView = (): void => {
    if (!visualizationRef.current) return;
    
    const svg = d3.select(visualizationRef.current).select("svg") as d3.Selection<SVGSVGElement, unknown, null, undefined>;
    const containerWidth = visualizationRef.current.clientWidth || 800;
    
    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.25, 5]);
    
    svg.call(zoom);
    
    const initialTransform = d3.zoomIdentity
      .translate(containerWidth / 2, 80)
      .scale(0.9);
    
    svg.call(zoom.transform, initialTransform);
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
            {algorithm && (<span> ({algorithm.toUpperCase()} algorithm)</span>)}
          </p>
        )}
      </div>
      
      {currentTrees.length > 1 && (
        <div className="p-4 bg-blue-50 border-b border-blue-100">
          <TreeSelector 
            count={currentTrees.length} 
            currentIndex={currentTreeIndex} 
            setCurrentIndex={setCurrentTreeIndex} 
            treeNames={currentTrees.map((_, i) => `Path ${i + 1}`)}
          />
        </div>
      )}
      
      {currentTrees.length > 0 && (
        <div className="p-3 bg-gray-50 border-b border-gray-200">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <button
                onClick={handleStartAnimation}
                disabled={isAnimating}
                className={`px-3 py-1.5 rounded text-sm font-medium flex items-center ${
                  isAnimating 
                    ? 'bg-gray-200 text-gray-500 cursor-not-allowed' 
                    : 'bg-blue-600 hover:bg-blue-700 text-white'
                }`}
              >
                <svg className="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                Animate Recipe
              </button>
              
              <button
                onClick={handleCenterView}
                className="px-2 py-1.5 rounded text-sm font-medium text-gray-600 bg-gray-200 hover:bg-gray-300 flex items-center"
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
                </svg>
                Center View
              </button>
            </div>
            
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
              <div className="flex items-center gap-1 ml-2">
                <input
                  type="checkbox"
                  id="autoCenter"
                  checked={autoCenter}
                  onChange={() => setAutoCenter(!autoCenter)}
                  className="h-3 w-3"
                />
                <label htmlFor="autoCenter" className="text-xs text-gray-600">Auto Center</label>
              </div>
              <div className="text-xs text-gray-500 flex items-center">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                {zoomLevel.toFixed(1)}×
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
              <p className="text-xs text-red-600 flex items-center">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
                {error}
              </p>
            </div>
          )}
        </div>
      )}
      
      <div 
        ref={visualizationRef} 
        className="w-full border-b border-gray-200 overflow-hidden bg-gradient-to-br from-gray-50 to-white relative flex items-center justify-center"
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
        <div className="p-6 bg-white border-t border-gray-100">
          <h3 className="text-lg font-semibold text-gray-800 mb-3 flex items-center">
            <svg className="w-5 h-5 mr-2 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            Recipe Details
          </h3>
          <div className="bg-gray-50 p-4 rounded-lg border border-gray-200">
            <TreeDetails 
              tree={currentTrees[currentTreeIndex]} 
              targetElement={targetElement}
              algorithm={algorithm}
            />
          </div>
        </div>
      )}
    </div>
  );
};

export default VisualizationPanel;
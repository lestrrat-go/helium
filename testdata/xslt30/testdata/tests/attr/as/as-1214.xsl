<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test with global xsl:variable that has @as="node()+", no @select and sequence constructor contains:
				-one node, a sequence of nodes of mixed kinds, nodes from a temporary tree  -->

   <t:variable name="temporary-tree">
      <a>
         <b att="one">hello</b>
      </a>
   </t:variable>

   <t:variable name="var1" as="node()+">
      <t:copy-of select="/doc/item"/>
   </t:variable>

   <t:variable name="var2" as="node()+">
      <t:copy-of select="//*"/>
      <t:copy-of select="/doc/item/@attrib"/>
      <t:copy-of select="/doc/processing-instruction()"/>
   </t:variable>

   <t:variable name="var3" as="node()+">
      <t:copy-of select="$temporary-tree//*"/>
      <t:copy-of select="$temporary-tree/a/b/@att"/>
      <t:copy-of select="$temporary-tree/a/comment()"/>
   </t:variable>

   <t:template match="/doc">
      <out>
         <e1>
            <t:value-of select="$var1 instance of node()+"/>
            <t:value-of select="$var1 instance of node()"/>
         </e1>
         <e2>
            <t:value-of select="$var2 instance of node()+"/>
            <t:value-of select="$var2 instance of node()"/>
         </e2>
         <e3>
            <t:value-of select="$var3 instance of node()+"/>
            <t:value-of select="$var3 instance of node()"/>
         </e3>
      </out>
   </t:template>
</t:transform>

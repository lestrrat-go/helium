<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test type of global xsl:variable selecting any node from input 
  				file or temporary tree, @as="node()". -->

   <t:variable name="temporary-tree">
      <a>
         <t:comment>I am a comment</t:comment>
         <b att="one">hello</b>
      </a>
   </t:variable>

   <t:variable name="var1" select="/" as="node()"/>

   <t:variable name="var2" select="/doc/item" as="node()"/>

   <t:variable name="var3" select="/doc/item/@attrib" as="node()"/>

   <t:variable name="var4" select="/doc/processing-instruction()" as="node()"/>

   <t:variable name="var5" select="/doc/item/text()" as="node()"/>

   <t:variable name="var6" select="/doc/item/namespace::*[1]" as="node()"/>

   <t:variable name="var7" select="/doc/item/comment()" as="node()"/>

   <t:variable name="var8" select="$temporary-tree" as="node()"/>

   <t:variable name="var9" select="$temporary-tree//a" as="node()"/>

   <t:variable name="var10" select="$temporary-tree/a/b/@att" as="node()"/>

   <t:variable name="var11" select="$temporary-tree/a/b/text()" as="node()"/>

   <t:variable name="var12" select="$temporary-tree/a/comment()" as="node()"/>

   <t:template match="/doc">
      <out>
         <e1>
            <t:value-of select="$var1 instance of node()"/>
         </e1>
         <e2>
            <t:value-of select="$var2 instance of node()"/>
         </e2>
         <e3>
            <t:value-of select="$var3 instance of node()"/>
         </e3>
         <e4>
            <t:value-of select="$var4 instance of node()"/>
         </e4>
         <e5>
            <t:value-of select="$var5 instance of node()"/>
         </e5>
         <e6>
            <t:value-of select="$var6 instance of node()"/>
         </e6>
         <e7>
            <t:value-of select="$var7 instance of node()"/>
         </e7>
         <e8>
            <t:value-of select="$var8 instance of node()"/>
         </e8>
         <e9>
            <t:value-of select="$var9 instance of node()"/>
         </e9>
         <e10>
            <t:value-of select="$var10 instance of node()"/>
         </e10>
         <e11>
            <t:value-of select="$var11 instance of node()"/>
         </e11>
         <e12>
            <t:value-of select="$var12 instance of node()"/>
         </e12>
      </out>
   </t:template>
</t:transform>

<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test with global xsl:variable that has @as=""element()*" and @select contains:
				- one element node, a sequence of element nodes from input file,
				an empty sequence obtained at run time, (), element nodes from a temporary tree.-->

   <t:variable name="temporary-tree">
      <a>
         <b att="one">hello</b>
      </a>
   </t:variable>

   <t:variable name="var1" select="/doc/item" as="element()*"/>

   <t:variable name="var2" select="/doc//*" as="element()*"/>

   <t:variable name="var3" select="/doc/elem" as="element()*"/>

   <t:variable name="var4" select="()" as="element()*"/>

   <t:variable name="var5" select="$temporary-tree//*" as="element()*"/>

   <t:template match="/doc">
      <out>
         <e1>
            <t:value-of select="$var1 instance of element()*"/>
            <t:value-of select="$var1 instance of element()"/>
         </e1>
         <e2>
            <t:value-of select="$var2 instance of element()*"/>
            <t:value-of select="$var2 instance of element()"/>
         </e2>
         <e3>
            <t:value-of select="$var3 instance of element()*"/>
            <t:value-of select="$var3 instance of element()"/>
         </e3>
         <e4>
            <t:value-of select="$var4 instance of element()*"/>
            <t:value-of select="$var4 instance of element()"/>
         </e4>
         <e5>
            <t:value-of select="$var5 instance of element()*"/>
            <t:value-of select="$var5 instance of element()"/>
         </e5>
      </out>
   </t:template>
</t:transform>

<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test with local xsl:param that has @as="element()+" and @select contains:
				-one element node, a sequence of element nodes from input file,
				element nodes from a temporary tree -->

   <t:variable name="temporary-tree">
      <a>
         <b att="one">hello</b>
      </a>
   </t:variable>

   <t:template match="/doc">
      <t:param name="par1" select="/doc/item" as="element()+"/>
      <t:param name="par2" select="/doc//*" as="element()+"/>
      <t:param name="par3" select="$temporary-tree//*" as="element()+"/>
      <out>
         <e1>
            <t:value-of select="$par1 instance of element()+"/>
            <t:value-of select="$par1 instance of element()"/>
         </e1>
         <e2>
            <t:value-of select="$par2 instance of element()+"/>
            <t:value-of select="$par2 instance of element()"/>
         </e2>
         <e3>
            <t:value-of select="$par3 instance of element()+"/>
            <t:value-of select="$par3 instance of element()"/>
         </e3>
      </out>
   </t:template>
</t:transform>

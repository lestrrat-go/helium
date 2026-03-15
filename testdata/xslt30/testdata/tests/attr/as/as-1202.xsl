<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://test.default.org" xmlns:t="http://www.w3.org/1999/XSL/Transform"
   exclude-result-prefixes="my" version="2.0">
   <!-- Purpose: Test type of global xsl:variable with @select selecting 
  				an element node from input file or temporary tree, @as="element(QName)". -->

   <t:variable name="temporary-tree">
      <a>
         <b att="one">hello</b>
      </a>
   </t:variable>

   <t:variable name="var1" select="/doc/item" as="element(item)"/>

   <t:variable name="var2" select="/doc/my:item1" as="element(my:item1)"/>

   <t:variable name="var3" select="$temporary-tree//a" as="element(a)"/>

   <t:template match="/doc">
      <out>
         <e1>
            <t:value-of select="$var1 instance of element(item)"/>
         </e1>
         <e2>
            <t:value-of select="$var2 instance of element(my:item1)"/>
         </e2>
         <e3>
            <t:value-of select="$var3 instance of element(a)"/>
         </e3>
      </out>
   </t:template>
</t:transform>

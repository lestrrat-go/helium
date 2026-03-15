<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">


   <!-- Purpose: Test case that uses use-when on an LRE in different levels (Parent-child, grandchild) 
      (The value of the attribute is true).  -->

   <xsl:template match="doc">
      <out>
         <e>
            <e1 xsl:use-when="1=1">
               <e1.1 xsl:use-when="true()">v1.1</e1.1>
               <e1.2 xsl:use-when="1=1">v1.2 <e1.2.1>v1.2.1</e1.2.1>> </e1.2>
            </e1>
         </e>
      </out>
   </xsl:template>

</xsl:stylesheet>

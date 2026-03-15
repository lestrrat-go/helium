<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://test.default.org" xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose: Test type of global xsl:variable with @select selecting an element node from 
  				input file or temporary tree, @as="element(QName, xs:untyped?)". -->

   <xslt:variable name="temporary-tree">
      <a>
         <b att="one">hello</b>
      </a>
   </xslt:variable>

   <xslt:variable name="var1" select="/doc/item" as="element(item, xs:untyped?)"/>

   <xslt:variable name="var2" select="/doc/my:item1" as="element(my:item1, xs:untyped?)"/>

   <xslt:variable name="var3" select="$temporary-tree//a" as="element(a, xs:untyped?)"/>

   <xslt:template match="/doc">
      <out>
         <e1>
            <xslt:value-of select="$var1 instance of element(item, xs:untyped?)"/>
         </e1>
         <e2>
            <xslt:value-of select="$var2 instance of element(my:item1, xs:untyped?)"/>
         </e2>
         <e3>
            <xslt:value-of select="$var3 instance of element(a, xs:untyped?)"/>
         </e3>
      </out>
   </xslt:template>
</xslt:transform>

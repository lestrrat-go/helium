<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.mytest.net"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with global xsl:variable that has @as="attribute()?" and @select contains:
				-one attribute node, an empty sequence obtained at run time, (),
				an attribute node from a temporary tree -->

   <xslt:variable name="temporary-tree">
      <a att="one">
         <a att="two">hello</a>
         <b att="three">hello</b>
      </a>
   </xslt:variable>

   <xslt:variable name="var1" select="/doc/@attrib" as="attribute()?"/>

   <xslt:variable name="var2" select="/doc/@attribute" as="attribute()?"/>

   <xslt:variable name="var3" select="()" as="attribute()?"/>

   <xslt:variable name="var4" select="$temporary-tree//b/@att" as="attribute()?"/>

   <xslt:template match="/doc">
      <out>
         <e1>
            <xslt:value-of select="$var1 instance of attribute()?"/>
            <xslt:value-of select="$var1 instance of attribute()"/>
         </e1>
         <e2>
            <xslt:value-of select="$var2 instance of attribute()?"/>
            <xslt:value-of select="$var2 instance of attribute()"/>
         </e2>
         <e3>
            <xslt:value-of select="$var3 instance of attribute()?"/>
            <xslt:value-of select="$var3 instance of attribute()"/>
         </e3>
         <e4>
            <xslt:value-of select="$var4 instance of attribute()?"/>
            <xslt:value-of select="$var4 instance of attribute()"/>
         </e4>
      </out>
   </xslt:template>
</xslt:transform>

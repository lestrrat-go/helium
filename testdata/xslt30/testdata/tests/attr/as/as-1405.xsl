<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.mytest.net"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with global xsl:variable that has @as="text()+" and @select contains:
				-one text node from input file, a sequence of text nodes from input file, 
				text nodes from a temporary tree  -->

   <xslt:variable name="temporary-tree">
      <a att="one">
         <a att="two">hello</a>
         <b att="three">hello</b>
      </a>
   </xslt:variable>

   <xslt:variable name="var1" select="/doc/item/text()" as="text()+"/>

   <xslt:variable name="var2" select="/doc//text()" as="text()+"/>

   <xslt:variable name="var3" select="$temporary-tree//text()" as="text()+"/>

   <xslt:template match="/doc">
      <out>
         <e1>
            <xslt:value-of select="$var1 instance of text()+"/>
            <xslt:value-of select="$var1 instance of text()"/>
         </e1>
         <e2>
            <xslt:value-of select="$var2 instance of text()+"/>
            <xslt:value-of select="$var2 instance of text()"/>
         </e2>
         <e3>
            <xslt:value-of select="$var3 instance of text()+"/>
            <xslt:value-of select="$var3 instance of text()"/>
         </e3>
      </out>
   </xslt:template>
</xslt:transform>

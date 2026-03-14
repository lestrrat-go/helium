<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.mytest.net"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with global xsl:variable that has @as="element(QName, xs:untyped)*" and its sequence constructor contains:
				- one element node, a sequence of element nodes from input file,
				an empty sequence obtained at run time, (), element nodes from a temporary tree.-->

   <xslt:variable name="temporary-tree">
      <a>
         <a att="one">hello</a>
      </a>
   </xslt:variable>

   <xslt:variable name="var1" as="element(my:elem, xs:untyped)*">
      <xslt:element name="my:elem">hello</xslt:element>
   </xslt:variable>

   <xslt:variable name="var2" as="element(item, xs:untyped)*">
      <xslt:copy-of select="/doc//*"/>
   </xslt:variable>

   <xslt:variable name="var3" as="element(elem, xs:untyped)*">
      <xslt:copy-of select="/doc/elem"/>
   </xslt:variable>

   <xslt:variable name="var4" as="element(elem, xs:untyped)*">
      <xslt:sequence select="()"/>
   </xslt:variable>

   <xslt:variable name="var5" as="element(a, xs:untyped)*">
      <xslt:copy-of select="$temporary-tree//*"/>
   </xslt:variable>

   <xslt:template match="/doc">
      <out>
         <e1>
            <xslt:value-of select="$var1 instance of element(my:elem, xs:untyped)*"/>
            <xslt:value-of select="$var1 instance of element(my:elem, xs:untyped)"/>
         </e1>
         <e2>
            <xslt:value-of select="$var2 instance of element(item, xs:untyped)*"/>
            <xslt:value-of select="$var2 instance of element(item, xs:untyped)"/>
         </e2>
         <e3>
            <xslt:value-of select="$var3 instance of element(elem, xs:untyped)*"/>
            <xslt:value-of select="$var3 instance of element(elem, xs:untyped)"/>
         </e3>
         <e4>
            <xslt:value-of select="$var4 instance of element(elem, xs:untyped)*"/>
            <xslt:value-of select="$var4 instance of element(elem, xs:untyped)"/>
         </e4>
         <e5>
            <xslt:value-of select="$var5 instance of element(a, xs:untyped)*"/>
            <xslt:value-of select="$var5 instance of element(a, xs:untyped)"/>
         </e5>
      </out>
   </xslt:template>
</xslt:transform>

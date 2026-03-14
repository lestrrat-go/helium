<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test type of global xsl:variable that has @as="item()*" and no @select, 
  				the sequence constructor contains one of:
				-nothing, LREs, xsl:value-of, xsl:sequence, sequence of xsl:element, xsl:document -->

   <xslt:variable name="temporary-tree">
      <a>
         <b att="one">hello</b>
      </a>
   </xslt:variable>

   <xslt:variable name="var1" as="item()*"> </xslt:variable>

   <xslt:variable name="var2" as="item()*">
      <i1>hello</i1>
      <i2>there</i2>
   </xslt:variable>

   <xslt:variable name="var3" as="item()*">
      <xslt:value-of select="'my'"/>
      <xslt:value-of select="$temporary-tree/a/b"/>
      <xslt:value-of select="'test'"/>
   </xslt:variable>

   <xslt:variable name="var4" as="item()*">
      <xslt:sequence select="(1,2,3)"/>
   </xslt:variable>

   <xslt:variable name="var5" as="item()*">
      <xslt:element name="el1">1</xslt:element>
      <xslt:element name="el2">2</xslt:element>
      <xslt:element name="el3">3</xslt:element>
      <xslt:element name="el4">4</xslt:element>
   </xslt:variable>

   <xslt:variable name="var6" as="item()*">
      <xslt:document>
         <item>hello</item>
      </xslt:document>
   </xslt:variable>

   <xslt:template match="/doc">
      <out>
         <e1>
            <xslt:value-of select="$var1 instance of item()*"/>
         </e1>
         <e2>
            <xslt:value-of select="$var2 instance of item()*"/>
         </e2>
         <e3>
            <xslt:value-of select="$var3 instance of item()*"/>
         </e3>
         <e4>
            <xslt:value-of select="$var4 instance of item()*"/>
         </e4>
         <e5>
            <xslt:value-of select="$var5 instance of item()*"/>
         </e5>
         <e6>
            <xslt:value-of select="$var6 instance of item()*"/>
         </e6>
      </out>
   </xslt:template>
</xslt:transform>

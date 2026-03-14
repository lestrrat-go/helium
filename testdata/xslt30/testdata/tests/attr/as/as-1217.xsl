<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.mytest.net"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with tunnel xsl:with-param that has @as=""element()*" and @select contains:
				-one element node, a sequence of element nodes from input file, an empty sequence 
				obtained at run time, (), element nodes from a temporary tree -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:variable name="temporary-tree">
      <a>
         <b att="one">hello</b>
      </a>
   </xslt:variable>

   <xslt:template match="/">
      <out>
         <xslt:apply-templates>
            <xslt:with-param name="par1" as="element()*" select="/doc/item" tunnel="yes"/>

            <xslt:with-param name="par2" as="element()*" select="/doc//*" tunnel="yes"/>

            <xslt:with-param name="par3" as="element()*" select="/doc/elem/*" tunnel="yes"/>

            <xslt:with-param name="par4" as="element()*" select="()" tunnel="yes"/>

            <xslt:with-param name="par5" as="element()*" select="$temporary-tree//*" tunnel="yes"/>

         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="doc">
      <xslt:apply-templates select="item"/>
   </xslt:template>

   <xslt:template match="item">
      <xslt:param name="par1" select="3.5" tunnel="yes"/>
      <xslt:param name="par2" select="3.5" tunnel="yes"/>
      <xslt:param name="par3" select="3.5" tunnel="yes"/>
      <xslt:param name="par4" select="3.5" tunnel="yes"/>
      <xslt:param name="par5" select="3.5" tunnel="yes"/>
      <par1>
         <xslt:value-of select="$par1"/>
         <xslt:value-of select="$par1 instance of element()*"/>
      </par1>
      <par2>
         <xslt:value-of select="$par2"/>
         <xslt:value-of select="$par2 instance of element()*"/>
      </par2>
      <par3>
         <xslt:value-of select="$par3"/>
         <xslt:value-of select="$par3 instance of element()*"/>
      </par3>
      <par4>
         <xslt:value-of select="$par4"/>
         <xslt:value-of select="$par4 instance of element()*"/>
      </par4>
      <par5>
         <xslt:value-of select="$par5"/>
         <xslt:value-of select="$par5 instance of element()*"/>
      </par5>
   </xslt:template>
</xslt:transform>

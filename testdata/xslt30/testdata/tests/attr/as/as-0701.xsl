<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://myexamplefunc.org"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test with tunnel xsl:param where value in @select is xs:untypedAtomic 
  				obtained at run time and @as=xs:double.  -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:function name="my:func1">
	     <xslt:param name="par"/>
	     <xslt:value-of select="$par instance of xs:double"/>
   </xslt:function>

   <xslt:template match="/">
      <xslt:param name="par1" select="//item1" as="xs:double"/>
      <out>
         <xslt:apply-templates>
	           <xslt:with-param name="par1" select="$par1" tunnel="yes"/>
         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="doc">
      <xslt:apply-templates select="item-list"/>
   </xslt:template>

   <xslt:template match="item-list">
      <xslt:param name="par1" tunnel="yes"/>
      <par1>
         <xslt:value-of select="$par1"/>
         <xslt:value-of select="$par1 instance of xs:double"/>
         <xslt:value-of select="my:func1($par1)"/>
      </par1>

   </xslt:template>
</xslt:transform>

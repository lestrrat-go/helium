<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test with several xs:untypedAtomic values as children of LRE in the sequence constructor of a tunnel xsl:with-param and 
  				@as=xs:float*. Verify the parameter is of type xs:float*. -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="/">
      <out>
         <xslt:apply-templates>
	           <xslt:with-param name="par1" as="xs:float*" tunnel="yes">
		             <xslt:sequence select="doc//item1"/>
	           </xslt:with-param>
         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="doc">
	     <xslt:apply-templates/>
   </xslt:template>

   <xslt:template match="item-list">
	     <xslt:apply-templates/>
   </xslt:template>

   <xslt:template match="item">
      <xslt:param name="par1" select="1.0" tunnel="yes"/>
      <par1>
         <xslt:value-of select="$par1"/>
         <xslt:value-of select="$par1 instance of xs:float*"/>
         <xslt:value-of select="$par1 instance of xs:float"/>
      </par1>
   </xslt:template>

   <xslt:template match="item1">
      <empty/>
   </xslt:template>
</xslt:transform>

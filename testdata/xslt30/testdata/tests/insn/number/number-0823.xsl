<?xml version="1.0" encoding="US-ASCII"?> <xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                 xmlns:k="urn:x-Ken"
                 version="2.0">
                 
                 <!-- error: context item for xsl:number is not a node -->
                 <?err XTTE0990?>

<xsl:output method="text"/>

<xsl:template match="/">
   <xsl:value-of select="k:showNumber(*/xsl:output)"/>
</xsl:template>

<xsl:function name="k:showNumber">
   <xsl:param name="cnode"/>
   <xsl:for-each select="1 to 5"><xsl:number/></xsl:for-each>
</xsl:function>

</xsl:stylesheet>

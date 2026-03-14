<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

<?spec xslt#patterns?>
    <!-- Purpose: xsl:number with count pattern using type pattern syntax -->
  

<xsl:template match="/">
 <out>
 	<xsl:apply-templates/>
 </out>
</xsl:template>

<xsl:template match="foo">
 <foo>
    <xsl:attribute name="number">
        <xsl:number level="any" count="doc[. instance of element()]"/>
    </xsl:attribute>
    <xsl:apply-templates/>
 </foo>
</xsl:template>



</xsl:stylesheet>

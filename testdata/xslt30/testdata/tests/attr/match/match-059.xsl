<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

<?spec xslt#patterns?>
    <!-- Purpose: Tilde syntax with node tests. -->

<xsl:template match="/">
 <out>
 	<xsl:apply-templates/>
 </out>
</xsl:template>

<xsl:template match="doc[. instance of element()]">
  <doc2><xsl:apply-templates/></doc2>
</xsl:template>

<xsl:template match="foo[. instance of element()]">
  <foo2><xsl:apply-templates/></foo2>
</xsl:template>


</xsl:stylesheet>

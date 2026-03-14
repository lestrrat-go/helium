<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

<?spec xslt#patterns?>
    <!-- Purpose: Union pattern incorrectly allowing node test or atomic value. -->

<xsl:template match="/">
 <out>
 	<xsl:apply-templates/>
 	<xsl:apply-templates select="28"/>
 </out>
</xsl:template>

<xsl:template match="element(doc)">
  <doc2><xsl:apply-templates/></doc2>
</xsl:template>

<xsl:template match="element(foo) | .[. instance of xs:integer]">
  <foo2/>
</xsl:template>


</xsl:stylesheet>

<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

<?spec xslt#patterns?>
    <!-- Purpose: the priority of the pattern .[. instance of xs:integer][. gt 0] is +1. -->
  <!-- Purpose: xsl:next-match when context item is not a node -->

<xsl:template name="main">
 <out>
 	<xsl:apply-templates select="17"/>
 </out>
</xsl:template>

<xsl:template match=".[. instance of xs:integer]" priority="0.999999">
  <third><xsl:next-match/></third>
</xsl:template>

<xsl:template match=".[. instance of xs:integer]" priority="1.0000001">
  <first><xsl:next-match/></first>
</xsl:template>

<xsl:template match=".[. instance of xs:integer][. gt 0]">
  <second><xsl:next-match/></second>
</xsl:template>

</xsl:stylesheet>

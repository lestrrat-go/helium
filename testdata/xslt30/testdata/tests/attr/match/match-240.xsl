<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

<?spec xslt#patterns?>
    <!-- Purpose: predicate pattern in which the type is not known statically.  -->

<xsl:template name="main">
 <out>
 	<xsl:apply-templates select="17"/>
 </out>
</xsl:template>


<xsl:template match=".[. instance of xs:integer]" priority="5">
  <first><xsl:next-match/></first>
</xsl:template>

<xsl:param name="N" required="yes"/>

<xsl:template match=".[. instance of xs:integer][$N]" priority="10">
  <second><xsl:next-match/></second>
</xsl:template>


</xsl:stylesheet>

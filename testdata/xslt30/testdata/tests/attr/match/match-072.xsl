<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

<?spec xslt#patterns?>
    <!-- Purpose: global variable reference as pattern -->
  
<xsl:variable name="v" select="//baz"/>

<xsl:template match="/">
<out>
 <xsl:apply-templates select="//*"/>
</out> 
</xsl:template>

<xsl:template match="$v">
  <ok/>
</xsl:template>

<xsl:template match="*"/>  

</xsl:stylesheet>

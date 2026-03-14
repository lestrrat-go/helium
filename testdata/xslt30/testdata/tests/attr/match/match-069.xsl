<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

<?spec xslt#patterns?>
    <!-- Purpose: parentless namespace node in pattern -->
  

<xsl:template match="/">
 <xsl:variable name="x" as="node()">
   <xsl:namespace name="x">http://www.example.com/ns</xsl:namespace>
 </xsl:variable>  
 <xsl:apply-templates select="$x"/>
</xsl:template>

<xsl:template match="namespace::*">
  <ns uri="{.}"/>
</xsl:template>  

</xsl:stylesheet>

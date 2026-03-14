<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

<?spec xslt#patterns?>
    <!-- Purpose: doc() function in pattern -->
  
<xsl:mode on-no-match="deep-skip"/>

<xsl:template match="/">
<out>
 <xsl:apply-templates select="doc('match1002.xml')"/>
</out> 
</xsl:template>

<xsl:template match="doc('match1002.xml')">
  <ok/>
</xsl:template>  

</xsl:stylesheet>

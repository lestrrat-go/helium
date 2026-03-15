<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

<?spec xslt#patterns?>
    <!-- Purpose: descendant axis in pattern with positional predicate -->
  

<xsl:template match="/">
<out>
 <xsl:apply-templates select="//*"/>
</out> 
</xsl:template>

<xsl:template match="(doc/descendant::foo)[2]">
  <ok att="{@att1}"/>
</xsl:template>

<xsl:template match="*"/>  

</xsl:stylesheet>

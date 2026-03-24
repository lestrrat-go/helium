<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
 <?spec fo#func-collection?>

<!-- PURPOSE: Get the default collection (by default this is an empty collection). -->

<xsl:template match="/">
<out>;
<result><xsl:copy-of select="collection()"/></result>;
</out>
</xsl:template>
       
</xsl:stylesheet>
